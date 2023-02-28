package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/keploy/go-sdk/integrations/kmux"
	"github.com/keploy/go-sdk/integrations/kredis"
	"github.com/keploy/go-sdk/keploy"
)

type redisCache struct {
	host    string
	db      int
	expires time.Duration
}

type data struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (cache *redisCache) getClient() redis.UniversalClient {
	client := redis.NewClient(&redis.Options{
		Addr:     cache.host,
		Password: "",
		DB:       cache.db,
	})
	return kredis.NewRedisClient(client)
}

func main() {
	fmt.Println("Application running on port 8080")
	cache := &redisCache{
		host:    "localhost:6379",
		db:      0,
		expires: time.Hour,
	}

	client := cache.getClient()
	msg, err := client.Ping(client.Context()).Result()
	if err != nil {
		fmt.Println("Error connecting to redis")
		return
	} else {
		fmt.Println(msg)
	}

	k := keploy.New(keploy.Config{
		App: keploy.AppConfig{
			Name: "my-app",
			Port: "8080",
		},
		Server: keploy.ServerConfig{
			URL: "http://localhost:6789/api",
		},
	})
	r := mux.NewRouter()
	r.Use(kmux.MuxMiddleware(k))
	r.HandleFunc("/data/{id}", cache.handleData).Methods("GET", "POST")

	http.ListenAndServe(":8080", r)
}

func (cache *redisCache) handleData(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	id := vars["id"]

	client := cache.getClient()

	if r.Method == "GET" {
		val, err := client.Get(r.Context(), id).Bytes()
		if err == redis.Nil {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var d data
		err = json.Unmarshal(val, &d)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)
	} else if r.Method == "POST" {
		var d data
		err := json.NewDecoder(r.Body).Decode(&d)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		val, err := json.Marshal(d)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = client.Set(r.Context(), id, val, cache.expires).Err()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(d)

		// line 119 to be uncommented
		// w.Write([]byte("Record saved"))

	}
}
