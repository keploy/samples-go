package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"loadtestapi/internal/store"
)

type Server struct {
	store  *store.Store
	logger *slog.Logger
}

type apiError struct {
	Error string `json:"error"`
}

func New(st *store.Store, logger *slog.Logger) http.Handler {
	s := &Server{
		store:  st,
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("POST /customers", s.createCustomer)
	mux.HandleFunc("POST /products", s.createProduct)
	mux.HandleFunc("POST /orders", s.createOrder)
	mux.HandleFunc("GET /orders/{id}", s.getOrder)
	mux.HandleFunc("GET /orders", s.searchOrders)
	mux.HandleFunc("GET /customers/{id}/summary", s.getCustomerSummary)
	mux.HandleFunc("GET /analytics/top-products", s.topProducts)
	mux.HandleFunc("POST /large-payloads", s.createLargePayload)
	mux.HandleFunc("GET /large-payloads/{id}", s.getLargePayload)
	mux.HandleFunc("DELETE /large-payloads/{id}", s.deleteLargePayload)

	return s.withRecover(s.withLogging(mux))
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "database unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) createCustomer(w http.ResponseWriter, r *http.Request) {
	var req store.CreateCustomerRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	customer, err := s.store.CreateCustomer(r.Context(), req)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, customer)
}

func (s *Server) createProduct(w http.ResponseWriter, r *http.Request) {
	var req store.CreateProductRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	product, err := s.store.CreateProduct(r.Context(), req)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req store.CreateOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	order, err := s.store.CreateOrder(r.Context(), req)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	order, err := s.store.GetOrder(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (s *Server) getCustomerSummary(w http.ResponseWriter, r *http.Request) {
	customerID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || customerID <= 0 {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "customer id must be a positive integer"})
		return
	}

	summary, err := s.store.GetCustomerSummary(r.Context(), customerID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) searchOrders(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	params := store.OrderSearchParams{
		Status:        query.Get("status"),
		MinTotalCents: parseInt(query.Get("min_total_cents"), 0),
		Limit:         parseInt(query.Get("limit"), 25),
		Offset:        parseInt(query.Get("offset"), 0),
	}

	if customerID := parseInt64(query.Get("customer_id"), 0); customerID > 0 {
		params.CustomerID = customerID
	}

	if value := query.Get("created_from"); value != "" {
		timestamp, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "created_from must use RFC3339"})
			return
		}
		params.CreatedFrom = &timestamp
	}

	if value := query.Get("created_through"); value != "" {
		timestamp, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "created_through must use RFC3339"})
			return
		}
		params.CreatedThrough = &timestamp
	}

	results, err := s.store.SearchOrders(r.Context(), params)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) topProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	days := parseInt(query.Get("days"), 30)
	limit := parseInt(query.Get("limit"), 10)

	results, err := s.store.TopProducts(r.Context(), days, limit)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) createLargePayload(w http.ResponseWriter, r *http.Request) {
	var req store.CreateLargePayloadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	record, err := s.store.CreateLargePayload(r.Context(), req)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) getLargePayload(w http.ResponseWriter, r *http.Request) {
	record, err := s.store.GetLargePayload(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) deleteLargePayload(w http.ResponseWriter, r *http.Request) {
	record, err := s.store.DeleteLargePayload(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) writeStoreError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, store.ErrValidation):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, store.ErrConflict), errors.Is(err, store.ErrInsufficientInventory):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
		message = err.Error()
	default:
		s.logger.Error("request failed", "error", err)
	}

	writeJSON(w, status, apiError{Error: message})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		start := time.Now()

		next.ServeHTTP(recorder, r)

		s.logger.Info(
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic recovered", "panic", recovered)
				writeJSON(w, http.StatusInternalServerError, apiError{Error: "internal server error"})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(`{"error":"internal server error"}`)
		statusCode = http.StatusInternalServerError
	}

	body = append(body, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseInt64(value string, fallback int64) int64 {
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

func contextWithTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}
