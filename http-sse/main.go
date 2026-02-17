package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

func main() {
	httpSrv := &http.Server{
		Addr:              "0.0.0.0:8000",
		Handler:           httpMux(),
		ReadHeaderTimeout: 50 * time.Second,
	}

	sseSrv := &http.Server{
		Addr:              "0.0.0.0:8047",
		Handler:           sseMux(),
		ReadHeaderTimeout: 50 * time.Second,
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		log.Printf("[HTTP] listening on %s\n", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[HTTP] ListenAndServe: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("[SSE ] listening on %s\n", sseSrv.Addr)
		if err := sseSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SSE] ListenAndServe: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(ctx)
	_ = sseSrv.Shutdown(ctx)

	wg.Wait()
	log.Println("bye")
}

/* ----------------------------- HTTP :8000 ----------------------------- */

func httpMux() http.Handler {
	mux := http.NewServeMux()

	// 1) health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"port": 8000,
		})
	})

	// 2) simple GET
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"msg":  "hello from http",
			"port": 8000,
		})
	})

	// 3) echo query param
	mux.HandleFunc("/api/echo", func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("msg")
		if msg == "" {
			msg = "empty"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"echo": msg,
			"port": 8000,
		})
	})

	// 4) add numbers
	mux.HandleFunc("/api/add", func(w http.ResponseWriter, r *http.Request) {
		qa := r.URL.Query().Get("a")
		qb := r.URL.Query().Get("b")
		a, _ := strconv.Atoi(qa)
		b, _ := strconv.Atoi(qb)
		writeJSON(w, http.StatusOK, map[string]any{
			"a":    a,
			"b":    b,
			"sum":  a + b,
			"port": 8000,
		})
	})

	// 5) time endpoint (dynamic, but ok for record; in replay you can mark noise if you want)
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"now":  time.Now().UTC().Format(time.RFC3339Nano),
			"port": 8000,
		})
	})

	// 6) fixed resource endpoint
	mux.HandleFunc("/api/resource/", func(w http.ResponseWriter, r *http.Request) {
		// path like /api/resource/123
		id := r.URL.Path[len("/api/resource/"):]
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "missing id"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":    id,
			"type":  "demo",
			"port":  8000,
			"fixed": true,
		})
	})

	// IMPORTANT: SSE routes MUST NOT be served on HTTP port.
	// If replay routes SSE traffic to HTTP :8000 => deterministic 404.
	mux.HandleFunc("/subscribe/student/events", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "SSE endpoint is NOT served on the HTTP port (:8000).",
			"hint":  "This simulates wrong port mapping during replay.",
		})
	})
	mux.HandleFunc("/subscribe/teacher/events", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "Teacher SSE endpoint is NOT served on the HTTP port (:8000).",
			"hint":  "This simulates wrong port mapping during replay.",
		})
	})

	// default
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error": "not found",
			"path":  r.URL.Path,
			"port":  8000,
		})
	})

	return mux
}

/* ----------------------------- SSE :8047 ------------------------------ */

func sseMux() http.Handler {
	mux := http.NewServeMux()

	// SSE #1 (student)
	mux.HandleFunc("/subscribe/student/events", func(w http.ResponseWriter, r *http.Request) {
		doubtID := r.URL.Query().Get("doubtId")
		if doubtID == "" {
			http.Error(w, "missing doubtId", http.StatusBadRequest)
			return
		}
		streamSSE(w, r, "student", doubtID)
	})

	// SSE #2 (teacher)
	mux.HandleFunc("/subscribe/teacher/events", func(w http.ResponseWriter, r *http.Request) {
		teacherID := r.URL.Query().Get("teacherId")
		if teacherID == "" {
			http.Error(w, "missing teacherId", http.StatusBadRequest)
			return
		}
		streamSSE(w, r, "teacher", teacherID)
	})

	return mux
}

func streamSSE(w http.ResponseWriter, r *http.Request, kind, id string) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	type msg struct {
		MessageID   string         `json:"message_id"`
		MessageType string         `json:"message_type"`
		Message     map[string]any `json:"message"`
	}

	now := time.Now().UnixMilli()

	// Event 1: TICKER
	ticker := []msg{{
		MessageID:   "ticker-1",
		MessageType: "TICKER",
		Message: map[string]any{
			"kind":      kind,
			"id":        id,
			"timestamp": now,
		},
	}}
	writeSSE(w, "TICKER", ticker)
	flusher.Flush()

	for i := 1; i <= 15; i++ {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		time.Sleep(1 * time.Second)
		now = time.Now().UnixMilli()

		message := []msg{{
			MessageID:   fmt.Sprintf("msg-%d", i),
			MessageType: "SYSTEM",
			Message: map[string]any{
				"timestamp": now,
				"title":     fmt.Sprintf("%s stream message %d", kind, i),
			},
		}}
		writeSSE(w, "message", message)
		flusher.Flush()
	}
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	b, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event:%s\n", event)
	fmt.Fprintf(w, "data:%s\n\n", string(b))
}

/* ------------------------------ helpers -------------------------------- */

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
