package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	defaultAddr            = ":8443"
	defaultCACert          = "/certs/ca.crt"
	defaultCertFile        = "/certs/server.crt"
	defaultKeyFile         = "/certs/server.key"
	minPayloadSizeBytes    = 50 * 1024
	maxPayloadSizeBytes    = 3 * 1024 * 1024
	responseSizeHeader     = "X-Response-Size-Bytes"
	contentTypeJSON        = "application/json"
	contentTypeOctetStream = "application/octet-stream"
)

type helloResponse struct {
	Message      string `json:"message"`
	ClientCommon string `json:"client_common_name"`
}

var errBodyTooLarge = errors.New("request body exceeds maximum size")

func main() {
	addr := getenv("SERVER_ADDR", defaultAddr)
	caCertPath := getenv("CA_CERT_FILE", defaultCACert)
	certFile := getenv("SERVER_CERT_FILE", defaultCertFile)
	keyFile := getenv("SERVER_KEY_FILE", defaultKeyFile)

	clientCAPool, err := loadCertPool(caCertPath)
	if err != nil {
		log.Fatalf("load client CA: %v", err)
	}

	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("load server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCAPool,
		Certificates: []tls.Certificate{serverCert},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}

		clientCert := r.TLS.PeerCertificates[0]
		resp := helloResponse{
			Message:      "mTLS handshake complete",
			ClientCommon: clientCert.Subject.CommonName,
		}

		w.Header().Set("Content-Type", contentTypeJSON)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("served /hello for client CN=%q", clientCert.Subject.CommonName)
	})

	mux.HandleFunc("/payload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed; use POST", http.StatusMethodNotAllowed)
			return
		}
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
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

		respBody := bytes.Repeat([]byte("r"), respSize)
		w.Header().Set("Content-Type", contentTypeOctetStream)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(respBody); err != nil {
			log.Printf("write /payload response failed: %v", err)
			return
		}

		clientCN := r.TLS.PeerCertificates[0].Subject.CommonName
		log.Printf("served /payload for client CN=%q req_size=%dB resp_size=%dB", clientCN, len(reqBody), len(respBody))
	})

	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	log.Printf("mTLS server listening on %s", addr)
	if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server stopped: %v", err)
	}
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

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
