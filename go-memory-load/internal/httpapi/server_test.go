package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestWriteJSONSetsContentLengthForLargeResponses(t *testing.T) {
	recorder := httptest.NewRecorder()
	payload := map[string]string{
		"payload": strings.Repeat("x", 4096),
	}

	writeJSON(recorder, http.StatusOK, payload)

	response := recorder.Result()
	t.Cleanup(func() {
		_ = response.Body.Close()
	})

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	if got := response.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("content-type = %q, want %q", got, "application/json")
	}

	if got := response.Header.Get("Content-Length"); got != strconv.Itoa(len(body)) {
		t.Fatalf("content-length = %q, want %q", got, strconv.Itoa(len(body)))
	}

	if len(body) <= 2048 {
		t.Fatalf("body length = %d, want > 2048 to cover large responses", len(body))
	}

	if body[len(body)-1] != '\n' {
		t.Fatalf("body should end with newline, got %q", body[len(body)-1:])
	}
}
