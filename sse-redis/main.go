// Package main implements a Redis-backed SSE server for Keploy testing.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisKey   = "sse-demo:messages"
	flushDelay = 120 * time.Millisecond
	boundary   = "keploy-stream-boundary"
)

var (
	rdb        *redis.Client
	startTime  = time.Now()
	instanceID = uuid.New().String()
)

type Message struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Category  string `json:"category"`
	CreatedAt string `json:"created_at"`
}

func main() {
	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	listenAddr := envOr("LISTEN_ADDR", ":8080")

	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("cannot reach Redis at %s: %v", redisAddr, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/messages", routeMessages)
	mux.HandleFunc("/events/sse", handleSSE)
	mux.HandleFunc("/events/ndjson", handleNDJSON)
	mux.HandleFunc("/events/multipart", handleMultipart)
	mux.HandleFunc("/events/plain", handlePlainText)
	mux.HandleFunc("/health", handleHealth)

	srv := &http.Server{Addr: listenAddr, Handler: mux}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
	}()

	log.Printf("listening on %s (redis=%s, instance=%s)", listenAddr, redisAddr, instanceID)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

func routeMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handleCreateMessage(w, r)
	case http.MethodDelete:
		handleClearMessages(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Text     string `json:"text"`
		Category string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Text == "" {
		http.Error(w, `"text" is required`, http.StatusBadRequest)
		return
	}

	msg := Message{
		ID:        uuid.New().String(),
		Text:      input.Text,
		Category:  input.Category,
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, "failed to serialize message", http.StatusInternalServerError)
		return
	}

	if err := rdb.RPush(r.Context(), redisKey, string(raw)).Err(); err != nil {
		http.Error(w, "redis write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          msg.ID,
		"text":        msg.Text,
		"category":    msg.Category,
		"created_at":  msg.CreatedAt,
		"server_time": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func handleClearMessages(w http.ResponseWriter, r *http.Request) {
	if err := rdb.Del(r.Context(), redisKey).Err(); err != nil {
		http.Error(w, "redis delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

// ---------------------------------------------------------------------------
// SSE endpoint — exercises exact doubtservice SSE patterns:
//   - TICKER events (instead of comments)
//   - named events with JSON array data
//   - no `id:` fields
// ---------------------------------------------------------------------------

func handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	messages, err := fetchMessages(r.Context())
	if err != nil {
		http.Error(w, "redis read failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// ticker event — doubtservice heartbeat pattern
	tickerData := []map[string]interface{}{
		{
			"message_id":      uuid.New().String(),
			"message_type":    "TICKER",
			"message_subtype": "",
			"message": map[string]interface{}{
				"timestamp": time.Now().UnixMilli(),
			},
			"message_data": nil,
		},
	}
	tickerBytes, _ := json.Marshal(tickerData)
	_, _ = fmt.Fprintf(w, "event:TICKER\ndata:%s\n\n", tickerBytes)
	flusher.Flush()

	for _, msg := range messages {
		// SYSTEM event (array payload)
		sysData := []map[string]interface{}{
			{
				"message_id":      msg.ID,
				"message_type":    "SYSTEM",
				"message_subtype": "DOUBT_RESOLVED", // random example subtype
				"message": map[string]interface{}{
					"timestamp": time.Now().UnixMilli(),
					"title":     msg.Text,
					"category":  msg.Category,
					"delay":     1000,
				},
				"message_data": nil,
			},
		}
		data, _ := json.Marshal(sysData)
		_, _ = fmt.Fprintf(w, "event:message\ndata:%s\n\n", data)
		flusher.Flush()
		time.Sleep(flushDelay)

		// secondary update event (array payload pattern)
		updateData := []map[string]interface{}{
			{
				"message_id":      msg.ID + "_update",
				"message_type":    "SYSTEM",
				"message_subtype": "FOOTER_UPDATE",
				"message": map[string]interface{}{
					"timestamp": time.Now().UnixMilli(),
					"footer": map[string]interface{}{
						"is_chat_enabled": false,
						"text":            msg.Category,
					},
				},
				"message_data": nil,
			},
		}
		updateJSON, _ := json.Marshal(updateData)
		_, _ = fmt.Fprintf(w, "event:message\ndata:%s\n\n", updateJSON)
		flusher.Flush()
		time.Sleep(flushDelay)
	}

	// stream finish mock message
	finalData := []map[string]interface{}{
		{
			"message_id":      uuid.New().String(),
			"message_type":    "SYSTEM",
			"message_subtype": "CLOSE_SSE_CONNECTION",
			"message": map[string]interface{}{
				"timestamp":      time.Now().UnixMilli(),
				"retry_required": false,
			},
			"message_data": nil,
		},
	}
	finalJSON, _ := json.Marshal(finalData)
	_, _ = fmt.Fprintf(w, "event:message\ndata:%s\n\n", finalJSON)
	flusher.Flush()
}

// ---------------------------------------------------------------------------
// NDJSON endpoint — one JSON object per line.
// ---------------------------------------------------------------------------

func handleNDJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	messages, err := fetchMessages(r.Context())
	if err != nil {
		http.Error(w, "redis read failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	requestID := uuid.New().String()
	w.Header().Set("Content-Type", "application/x-ndjson")

	for i, msg := range messages {
		event := buildStreamPayload(msg, requestID)
		data, _ := json.Marshal(event)
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n"))
		flusher.Flush()
		if i < len(messages)-1 {
			time.Sleep(flushDelay)
		}
	}
}

// ---------------------------------------------------------------------------
// Multipart endpoint — multipart/x-mixed-replace with JSON parts.
// ---------------------------------------------------------------------------

func handleMultipart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	messages, err := fetchMessages(r.Context())
	if err != nil {
		http.Error(w, "redis read failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	requestID := uuid.New().String()
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+boundary)

	for i, msg := range messages {
		event := buildStreamPayload(msg, requestID)
		data, _ := json.Marshal(event)

		_, _ = fmt.Fprintf(w, "--%s\r\nContent-Type: application/json\r\n\r\n%s\r\n", boundary, data)
		flusher.Flush()
		if i < len(messages)-1 {
			time.Sleep(flushDelay)
		}
	}
	_, _ = fmt.Fprintf(w, "--%s--\r\n", boundary)
	flusher.Flush()
}

// ---------------------------------------------------------------------------
// Plain text endpoint — chunked line-based stream.
// ---------------------------------------------------------------------------

func handlePlainText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	messages, err := fetchMessages(r.Context())
	if err != nil {
		http.Error(w, "redis read failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")

	for i, msg := range messages {
		line := fmt.Sprintf("[%s] %s: %s\n", strings.ToUpper(msg.Category), msg.ID, msg.Text)
		_, _ = w.Write([]byte(line))
		flusher.Flush()
		if i < len(messages)-1 {
			time.Sleep(flushDelay)
		}
	}
}

// ---------------------------------------------------------------------------
// Health endpoint
// ---------------------------------------------------------------------------

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"server_time":    time.Now().UTC().Format(time.RFC3339Nano),
		"uptime_seconds": int(time.Since(startTime).Seconds()),
		"instance_id":    instanceID,
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fetchMessages(ctx context.Context) ([]Message, error) {
	raw, err := rdb.LRange(ctx, redisKey, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	msgs := make([]Message, 0, len(raw))
	for _, r := range raw {
		var m Message
		if json.Unmarshal([]byte(r), &m) == nil {
			msgs = append(msgs, m)
		}
	}
	return msgs, nil
}

// buildStreamPayload produces the JSON object sent in SSE data, NDJSON lines,
// and multipart parts. Fields from Redis are stable across record/replay
// (mocked). Runtime fields (delivered_at, request_id) change every run.
func buildStreamPayload(msg Message, requestID string) map[string]interface{} {
	return map[string]interface{}{
		"id":           msg.ID,
		"text":         msg.Text,
		"category":     msg.Category,
		"created_at":   msg.CreatedAt,
		"delivered_at": time.Now().UTC().Format(time.RFC3339Nano),
		"request_id":   requestID,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
