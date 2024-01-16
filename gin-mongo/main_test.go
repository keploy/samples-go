package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/v2/keploy"
)

func setup(t *testing.T) {
	err := keploy.New(keploy.Config{
		Name:             "TestPutURL",
		Mode:             keploy.MODE_RECORD,
		Path:             "<path for storing stubs>",
		Delay:            10,
	})
	if err != nil {
		t.Fatalf("error while running keploy: %v", err)
	}
	dbName, collection := "keploy", "url-shortener"
	client, err := New("localhost:27017", dbName)
	if err != nil {
		panic("Failed to initialize MongoDB: " + err.Error())
	}
	db := client.Database(dbName)
	col = db.Collection(collection)
}



func TestPutURL(t *testing.T) {

	defer keploy.KillProcessOnPort()
	setup(t)

	r := gin.Default()
	r.GET("/:param", getURL)
	r.POST("/url", putURL)

	data := map[string]string{
		"url": "https://www.example.com",
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("rfe: %v\n", err)
	}

	req, err := http.NewRequest(http.MethodPost, "/url", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatalf("Couldn't create request: %v\n", err)
	}
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Checking if the URL was successfully shortened and stored
	if w.Code != http.StatusOK {
		t.Fatalf("Exect HTTP 200 OK, but got %v", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v\n", err)
	}
	fmt.Println("response-url" + response["url"].(string))

	if response["url"] == nil || response["ts"] == nil {
		t.Fatalf("Response did not contain expected fields")
	}
}