package main_test

import (
    "encoding/json"
    "log"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/gin-gonic/gin"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/postgres"
    app "github.com/keploy/samples-go/go-jwt" // Replace with the correct path to your app package
)

// setupTestRouter initializes the router for testing
func setupTestRouter() *gin.Engine {
    router := gin.Default()

    dsn := "postgres://postgres:postgres@localhost:5432/test_db?sslmode=disable"
    db, err := gorm.Open("postgres", dsn)
    if err != nil {
        log.Fatal("Failed to connect to test database:", err)
    }
    app.Db = db
    db.AutoMigrate(&app.User{})

    router.GET("/health", app.HealthCheckHandler)
    router.GET("/generate-token", app.GenerateTokenHandler)
    router.GET("/check-token", app.CheckTokenHandler)

    return router
}

func TestHealthCheckHandler(t *testing.T) {
    router := setupTestRouter()

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/health", nil)
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.JSONEq(t, `{"status": "healthy"}`, w.Body.String())
}

func TestGenerateTokenHandler(t *testing.T) {
    router := setupTestRouter()

    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/generate-token", nil)
    router.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)

    var response map[string]string
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.Nil(t, err)

    token, ok := response["token"]
    assert.True(t, ok)
    assert.NotEmpty(t, token)
}

func TestCheckTokenHandler(t *testing.T) {
    router := setupTestRouter()

    generateW := httptest.NewRecorder()
    generateReq, _ := http.NewRequest("GET", "/generate-token", nil)
    router.ServeHTTP(generateW, generateReq)

    var generateResponse map[string]string
    err := json.Unmarshal(generateW.Body.Bytes(), &generateResponse)
    assert.Nil(t, err)

    token := generateResponse["token"]

    checkW := httptest.NewRecorder()
    checkReq, _ := http.NewRequest("GET", "/check-token?token="+token, nil)
    router.ServeHTTP(checkW, checkReq)

    assert.Equal(t, http.StatusOK, checkW.Code)

    var checkResponse map[string]string
    err = json.Unmarshal(checkW.Body.Bytes(), &checkResponse)
    assert.Nil(t, err)

    assert.Equal(t, "example_user", checkResponse["username"])
}