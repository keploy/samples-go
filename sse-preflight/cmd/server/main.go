package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	httpPort := flag.Int("http-port", 8000, "normal HTTP port (non-SSE)")
	ssePort := flag.Int("sse-port", 8047, "SSE port")
	flag.Parse()

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	// Intentionally do NOT register /subscribe/student/events (or "/") on :8000.
	// This allows us to reproduce the 404 when Keploy replays the SSE preflight on the wrong port.

	sseMux := http.NewServeMux()
	sseMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	sseMux.HandleFunc("/subscribe/student/events", handleEvents)

	httpSrv := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", *httpPort), Handler: httpMux}
	sseSrv := &http.Server{Addr: fmt.Sprintf("0.0.0.0:%d", *ssePort), Handler: sseMux}

	type serverErr struct {
		name string
		addr string
		err  error
	}

	exitCode := 0
	errCh := make(chan serverErr, 2)
	go func() {
		log.Printf("HTTP listening on %s", httpSrv.Addr)
		errCh <- serverErr{name: "HTTP", addr: httpSrv.Addr, err: httpSrv.ListenAndServe()}
	}()
	go func() {
		log.Printf("SSE listening on %s", sseSrv.Addr)
		errCh <- serverErr{name: "SSE", addr: sseSrv.Addr, err: sseSrv.ListenAndServe()}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stop:
		log.Printf("signal received: %s", sig)
	case server := <-errCh:
		if server.err != nil && server.err != http.ErrServerClosed {
			log.Printf("%s listener error on %s: %v", server.name, server.addr, server.err)
			log.Printf("hint: check for port conflicts/permissions, then retry")
			exitCode = 1
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	_ = sseSrv.Shutdown(ctx)

	return exitCode
}

func writeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "7200")
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	writeCORS(w)

	// CORS preflight: respond successfully, but do NOT set text/event-stream.
	// This is key to reproducing the Keploy issue: the test case won't be detected as SSE.
	if r.Method == http.MethodOptions {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	doubtID := r.URL.Query().Get("doubtId")
	if doubtID == "" {
		doubtID = "missing"
	}

	ctx := r.Context()
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if _, err := fmt.Fprintf(w, "event: message\ndata: {\"doubtId\":\"%s\",\"n\":%d}\n\n", doubtID, i); err != nil {
			return
		}
		flusher.Flush()

		if i < 2 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
}
