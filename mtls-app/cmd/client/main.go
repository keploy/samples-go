package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	defaultServerURL         = "https://mtls-server:8443/hello"
	defaultBigPayloadURL     = "https://mtls-server:8443/payload"
	defaultCACert            = "/certs/ca.crt"
	defaultCertFile          = "/certs/client.crt"
	defaultKeyFile           = "/certs/client.key"
	minPayloadSizeBytes      = 50 * 1024
	maxPayloadSizeBytes      = 3 * 1024 * 1024
	responseSizeHeader       = "X-Response-Size-Bytes"
	modeDefault              = "default"
	modeBigPayload           = "bigpayload"
	contentTypeOctetStream   = "application/octet-stream"
	contentTypeJSON          = "application/json"
	defaultSingleRunAttempts = 12
	defaultAPIRetryAttempts  = 3
)

type upstreamResponse struct {
	status      string
	statusCode  int
	contentType string
	body        []byte
}

type upstreamRequest struct {
	method      string
	url         string
	body        []byte
	contentType string
	headers     map[string]string
}

var errBodyTooLarge = errors.New("request body exceeds maximum size")

func main() {
	serverURL := getenv("SERVER_URL", defaultServerURL)
	bigPayloadURL := getenv("BIGPAYLOAD_SERVER_URL", deriveBigPayloadURL(serverURL))
	caCertPath := getenv("CA_CERT_FILE", defaultCACert)
	certFile := getenv("CLIENT_CERT_FILE", defaultCertFile)
	keyFile := getenv("CLIENT_KEY_FILE", defaultKeyFile)
	apiAddr := os.Getenv("CLIENT_API_ADDR")
	clientMode := getenv("CLIENT_MODE", modeDefault)
	if clientMode != modeDefault && clientMode != modeBigPayload {
		log.Printf("unknown CLIENT_MODE=%q, falling back to %q", clientMode, modeDefault)
		clientMode = modeDefault
	}

	rootCAs, err := loadCertPool(caCertPath)
	if err != nil {
		log.Fatalf("load CA cert: %v", err)
	}

	clientCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("load client certificate: %v", err)
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:   tls.VersionTLS12,
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{clientCert},
				ServerName:   "mtls-server",
			},
		},
	}

	if apiAddr != "" {
		startAPI(apiAddr, serverURL, bigPayloadURL, clientMode, httpClient)
		return
	}

	req := upstreamRequest{
		method: http.MethodGet,
		url:    serverURL,
	}
	if clientMode == modeBigPayload {
		body := bytes.Repeat([]byte("b"), minPayloadSizeBytes)
		req = upstreamRequest{
			method:      http.MethodPost,
			url:         bigPayloadURL,
			body:        body,
			contentType: contentTypeOctetStream,
			headers: map[string]string{
				responseSizeHeader: strconv.Itoa(minPayloadSizeBytes),
			},
		}
	}

	resp, err := requestWithRetries(context.Background(), httpClient, req, defaultSingleRunAttempts)
	if err != nil {
		log.Fatalf("request failed after retries: %v", err)
	}

	fmt.Printf("response status: %s\n", resp.status)
	if clientMode == modeBigPayload {
		fmt.Printf("response body size: %d bytes\n", len(resp.body))
		return
	}
	fmt.Printf("response body: %s\n", string(resp.body))
}

func startAPI(addr, serverURL, bigPayloadURL, clientMode string, httpClient *http.Client) {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		resp, err := requestWithRetries(r.Context(), httpClient, upstreamRequest{
			method: http.MethodGet,
			url:    serverURL,
		}, defaultAPIRetryAttempts)
		if err != nil {
			http.Error(w, fmt.Sprintf("upstream request failed: %v", err), http.StatusBadGateway)
			return
		}

		contentType := resp.contentType
		if contentType == "" {
			contentType = contentTypeJSON
		}

		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(resp.statusCode)
		_, _ = w.Write(resp.body)
	})

	if clientMode == modeBigPayload {
		mux.HandleFunc("/bigpayload", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed; use POST", http.StatusMethodNotAllowed)
				return
			}

			reqBody, err := readBoundedBody(r.Body, maxPayloadSizeBytes)
			if err != nil {
				status := http.StatusBadRequest
				if errors.Is(err, errBodyTooLarge) {
					status = http.StatusRequestEntityTooLarge
				}
				http.Error(w, err.Error(), status)
				return
			}

			if err := validatePayloadSize(len(reqBody), "request"); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			respSize, err := resolveResponseSize(r, len(reqBody))
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if contentType == "" {
				contentType = contentTypeOctetStream
			}

			resp, err := requestWithRetries(r.Context(), httpClient, upstreamRequest{
				method:      http.MethodPost,
				url:         bigPayloadURL,
				body:        reqBody,
				contentType: contentType,
				headers: map[string]string{
					responseSizeHeader: strconv.Itoa(respSize),
				},
			}, defaultAPIRetryAttempts)
			if err != nil {
				http.Error(w, fmt.Sprintf("upstream payload request failed: %v", err), http.StatusBadGateway)
				return
			}

			upstreamContentType := resp.contentType
			if upstreamContentType == "" {
				upstreamContentType = contentTypeOctetStream
			}

			w.Header().Set("Content-Type", upstreamContentType)
			w.WriteHeader(resp.statusCode)
			_, _ = w.Write(resp.body)
			log.Printf("client /bigpayload request complete req_size=%dB resp_size=%dB", len(reqBody), len(resp.body))
		})
	}

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("mTLS client API listening on %s", addr)
	log.Printf("GET /hello -> calls %s over mTLS", serverURL)
	if clientMode == modeBigPayload {
		log.Printf("POST /bigpayload -> calls %s over mTLS with payloads [%dB, %dB]", bigPayloadURL, minPayloadSizeBytes, maxPayloadSizeBytes)
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("client API stopped: %v", err)
	}
}

func requestWithRetries(ctx context.Context, httpClient *http.Client, reqCfg upstreamRequest, maxAttempts int) (*upstreamResponse, error) {
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := requestOnce(ctx, httpClient, reqCfg)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		log.Printf("request attempt %d failed: %v", attempt, err)

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}
	}

	return nil, lastErr
}

func requestOnce(ctx context.Context, httpClient *http.Client, reqCfg upstreamRequest) (*upstreamResponse, error) {
	reqBody := bytes.NewReader(reqCfg.body)
	req, err := http.NewRequestWithContext(ctx, reqCfg.method, reqCfg.url, reqBody)
	if err != nil {
		return nil, err
	}
	if reqCfg.contentType != "" {
		req.Header.Set("Content-Type", reqCfg.contentType)
	}
	for key, value := range reqCfg.headers {
		req.Header.Set(key, value)
	}

	start := time.Now()
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	log.Printf("upstream request complete method=%s url=%s status=%d req_size=%dB resp_size=%dB duration_ms=%d",
		reqCfg.method, reqCfg.url, resp.StatusCode, len(reqCfg.body), len(body), time.Since(start).Milliseconds())

	return &upstreamResponse{
		status:      resp.Status,
		statusCode:  resp.StatusCode,
		contentType: resp.Header.Get("Content-Type"),
		body:        body,
	}, nil
}

func loadCertPool(path string) (*x509.CertPool, error) {
	certPEM, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		return nil, errors.New("invalid PEM data for CA certificate")
	}

	return pool, nil
}

func readBoundedBody(r io.Reader, maxBytes int) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	if len(body) > maxBytes {
		return nil, errBodyTooLarge
	}
	return body, nil
}

func validatePayloadSize(size int, kind string) error {
	if size < minPayloadSizeBytes || size > maxPayloadSizeBytes {
		return fmt.Errorf("%s payload size must be between %d and %d bytes, got %d", kind, minPayloadSizeBytes, maxPayloadSizeBytes, size)
	}
	return nil
}

func resolveResponseSize(r *http.Request, fallback int) (int, error) {
	sizeRaw := r.URL.Query().Get("response_size_bytes")
	if sizeRaw == "" {
		sizeRaw = r.Header.Get(responseSizeHeader)
	}
	if sizeRaw == "" {
		return fallback, nil
	}

	size, err := strconv.Atoi(sizeRaw)
	if err != nil {
		return 0, fmt.Errorf("invalid response size %q", sizeRaw)
	}
	if err := validatePayloadSize(size, "response"); err != nil {
		return 0, err
	}
	return size, nil
}

func deriveBigPayloadURL(serverURL string) string {
	parsed, err := url.Parse(serverURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return defaultBigPayloadURL
	}
	parsed.Path = "/payload"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
