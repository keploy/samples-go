package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type app struct {
	sqlDB   *sql.DB
	mongoDB *mongo.Database
	started time.Time
}

type user struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type auditEvent struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type      string             `bson:"type" json:"type"`
	Label     string             `bson:"label,omitempty" json:"label,omitempty"`
	UserID    int64              `bson:"user_id,omitempty" json:"user_id,omitempty"`
	Name      string             `bson:"name,omitempty" json:"name,omitempty"`
	Email     string             `bson:"email,omitempty" json:"email,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type touchRequest struct {
	Label string `json:"label"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sqlDB, err := connectPostgres(ctx, env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pcapdemo?sslmode=disable"))
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer sqlDB.Close()

	mongoClient, mongoDB, err := connectMongo(ctx, env("MONGO_URI", "mongodb://localhost:27017"), env("MONGO_DATABASE", "pcapdemo"))
	if err != nil {
		log.Fatalf("mongo: %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(shutdownCtx); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}()

	a := &app{
		sqlDB:   sqlDB,
		mongoDB: mongoDB,
		started: time.Now(),
	}

	handler := logRequests(a.routes())
	httpServer := &http.Server{
		Addr:              env("HTTP_ADDR", ":8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	httpsServer := &http.Server{
		Addr:              env("HTTPS_ADDR", ":8443"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() {
		log.Printf("plain HTTP listening on %s", httpServer.Addr)
		errCh <- ignoreServerClosed(httpServer.ListenAndServe())
	}()
	go func() {
		tlsConfig, source, err := loadTLSConfig()
		if err != nil {
			errCh <- fmt.Errorf("tls config: %w", err)
			return
		}
		listener, err := tls.Listen("tcp", httpsServer.Addr, tlsConfig)
		if err != nil {
			errCh <- err
			return
		}
		log.Printf("TLS HTTPS listening on %s (%s)", httpsServer.Addr, source)
		errCh <- ignoreServerClosed(httpsServer.Serve(listener))
	}()

	select {
	case <-ctx.Done():
		log.Println("shutdown requested")
	case err := <-errCh:
		if err != nil {
			log.Printf("server error: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	if err := httpsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("https shutdown: %v", err)
	}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleRoot)
	mux.HandleFunc("/healthz", a.handleHealth)
	mux.HandleFunc("/users", a.handleUsers)
	mux.HandleFunc("/touch", a.handleTouch)
	return mux
}

func (a *app) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name": "pcap-validator",
		"routes": []string{
			"GET /healthz",
			"GET /users",
			"POST /users",
			"POST /touch",
		},
	})
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	sqlErr := a.sqlDB.PingContext(ctx)
	mongoErr := a.mongoDB.Client().Ping(ctx, readpref.Primary())

	status := http.StatusOK
	if sqlErr != nil || mongoErr != nil {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, map[string]any{
		"ok":          status == http.StatusOK,
		"uptime_ms":   time.Since(a.started).Milliseconds(),
		"sql_error":   errorString(sqlErr),
		"mongo_error": errorString(mongoErr),
	})
}

func (a *app) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.listUsers(w, r)
	case http.MethodPost:
		a.createUser(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *app) createUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	name := strings.TrimSpace(req.Name)
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if name == "" || email == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var created user
	row := a.sqlDB.QueryRowContext(ctx, `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, name, email, created_at
	`, name, email)
	if err := row.Scan(&created.ID, &created.Name, &created.Email, &created.CreatedAt); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	event := auditEvent{
		Type:      "user_created",
		UserID:    created.ID,
		Name:      created.Name,
		Email:     created.Email,
		CreatedAt: time.Now().UTC(),
	}
	inserted, err := a.mongoDB.Collection("audit_events").InsertOne(ctx, event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user":     created,
		"audit_id": inserted.InsertedID,
	})
}

func (a *app) listUsers(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := a.sqlDB.QueryContext(ctx, `
		SELECT id, name, email, created_at
		FROM users
		ORDER BY id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	users := make([]user, 0)
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	events, err := a.latestAuditEvents(ctx, 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":  users,
		"events": events,
	})
}

func (a *app) handleTouch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req touchRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	label := strings.TrimSpace(req.Label)
	if label == "" {
		label = fmt.Sprintf("touch-%d", time.Now().UnixNano())
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var sqlEventID int64
	if err := a.sqlDB.QueryRowContext(ctx, `
		INSERT INTO sql_events (label)
		VALUES ($1)
		RETURNING id
	`, label).Scan(&sqlEventID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	inserted, err := a.mongoDB.Collection("audit_events").InsertOne(ctx, auditEvent{
		Type:      "manual_touch",
		Label:     label,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"label":          label,
		"sql_event_id":   sqlEventID,
		"mongo_event_id": inserted.InsertedID,
	})
}

func (a *app) latestAuditEvents(ctx context.Context, limit int64) ([]auditEvent, error) {
	cursor, err := a.mongoDB.Collection("audit_events").Find(
		ctx,
		bson.D{},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	events := make([]auditEvent, 0)
	for cursor.Next(ctx) {
		var event auditEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func connectPostgres(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := retry(ctx, "postgres ping", func(ctx context.Context) error {
		return db.PingContext(ctx)
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migratePostgres(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migratePostgres(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS sql_events (
			id BIGSERIAL PRIMARY KEY,
			label TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	return err
}

func connectMongo(ctx context.Context, uri string, databaseName string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}
	if err := retry(ctx, "mongo ping", func(ctx context.Context) error {
		return client.Ping(ctx, readpref.Primary())
	}); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}

	db := client.Database(databaseName)
	index := mongo.IndexModel{
		Keys: bson.D{{Key: "created_at", Value: -1}},
	}
	if _, err := db.Collection("audit_events").Indexes().CreateOne(ctx, index); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, nil, err
	}
	return client, db, nil
}

func retry(ctx context.Context, label string, fn func(context.Context) error) error {
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		lastErr = fn(attemptCtx)
		cancel()
		if lastErr == nil {
			return nil
		}
		log.Printf("%s failed (attempt %d/30): %v", label, attempt, lastErr)

		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func loadTLSConfig() (*tls.Config, string, error) {
	certFile := strings.TrimSpace(os.Getenv("TLS_CERT_FILE"))
	keyFile := strings.TrimSpace(os.Getenv("TLS_KEY_FILE"))
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			return nil, "", errors.New("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, "", err
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}, "configured certificate", nil
	}

	cert, err := generateSelfSignedCertificate()
	if err != nil {
		return nil, "", err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, "ephemeral self-signed certificate", nil
}

func generateSelfSignedCertificate() (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"pcap-demo"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	return tls.X509KeyPair(certPEM, keyPEM)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return fallback
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %s", r.Proto, r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}

func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
