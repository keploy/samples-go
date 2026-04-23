// Minimal net/http + database/sql + MySQL sample used by Keploy's
// end-to-end pipeline. Exercises the full MySQL connection-and-command
// phase so regressions in the replay proxy (e.g. a zero read deadline
// in the command-phase loop) surface as the Go MySQL driver blocking
// inside db.Ping(), which prevents the HTTP server from binding.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var db *sql.DB

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	host := envDefault("MYSQL_HOST", "mysql")
	port := envDefault("MYSQL_PORT", "3306")
	user := envDefault("MYSQL_USER", "root")
	pass := envDefault("MYSQL_PASSWORD", "password")
	dbname := envDefault("MYSQL_DATABASE", "testdb")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)
	log.Printf("Connecting to MySQL at %s:%s/%s", host, port, dbname)

	var err error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("waiting for mysql (attempt %d): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("could not connect to mysql after retries: %v", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100),
		email VARCHAR(100)
	)`); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		if _, err := db.Exec("INSERT INTO users (name,email) VALUES ('alice','alice@example.com'), ('bob','bob@example.com')"); err != nil {
			log.Fatalf("failed to seed: %v", err)
		}
	}

	http.HandleFunc("/users", getUsers)
	http.HandleFunc("/users/add", addUser)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(users)
}

func addUser(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	email := r.URL.Query().Get("email")
	if name == "" || email == "" {
		http.Error(w, "name and email required", http.StatusBadRequest)
		return
	}
	res, err := db.Exec("INSERT INTO users (name,email) VALUES (?,?)", name, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(User{ID: int(id), Name: name, Email: email})
}
