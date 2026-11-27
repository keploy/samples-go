package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEvaluateTimeSkewMillisecondPrecision(t *testing.T) {
	// Recorded at .968s into a second — previously truncated by `date +%s`,
	// which lost 968ms of margin before the request even reached the server.
	recorded := time.Date(2026, 11, 27, 7, 19, 15, 968_000_000, time.UTC)
	clientMs := recorded.UnixMilli()

	// Crossing a second boundary 50ms later must report 50ms, not 1s.
	serverVal, diff, ok := evaluateTimeSkew(recorded.Add(50*time.Millisecond), clientMs)
	if !ok || diff != 50*time.Millisecond || serverVal != clientMs+50 {
		t.Fatalf("50ms across boundary: server=%d diff=%v ok=%v, want ok=true diff=50ms", serverVal, diff, ok)
	}

	// Exactly 1000ms skew is within the 1-second tolerance.
	_, diff, ok = evaluateTimeSkew(recorded.Add(time.Second), clientMs)
	if !ok || diff != time.Second {
		t.Fatalf("1000ms skew: diff=%v ok=%v, want ok=true diff=1s", diff, ok)
	}

	// 1001ms skew exceeds the 1-second tolerance (tolerance was not raised).
	_, diff, ok = evaluateTimeSkew(recorded.Add(1001*time.Millisecond), clientMs)
	if ok || diff != 1001*time.Millisecond {
		t.Fatalf("1001ms skew: diff=%v ok=%v, want ok=false diff=1.001s", diff, ok)
	}

	// Negative skew within 1s passes; beyond 1s fails.
	if _, _, ok := evaluateTimeSkew(recorded.Add(-900*time.Millisecond), clientMs); !ok {
		t.Fatal("-900ms skew should pass")
	}
	if _, _, ok := evaluateTimeSkew(recorded.Add(-1001*time.Millisecond), clientMs); ok {
		t.Fatal("-1001ms skew should fail")
	}
}

func TestEvaluateTimeSkewLegacySeconds(t *testing.T) {
	base := time.Date(2026, 11, 27, 7, 19, 15, 500_000_000, time.UTC)
	clientSec := base.Unix()

	if _, diff, ok := evaluateTimeSkew(base.Add(time.Second), clientSec); !ok || diff != time.Second {
		t.Fatalf("legacy +1s: diff=%v ok=%v, want ok=true diff=1s", diff, ok)
	}
	if _, diff, ok := evaluateTimeSkew(base.Add(2*time.Second), clientSec); ok || diff != 2*time.Second {
		t.Fatalf("legacy +2s: diff=%v ok=%v, want ok=false diff=2s", diff, ok)
	}
}

func TestCheckTimeHandlerHTTP(t *testing.T) {
	// Missing ts -> 400
	rec := httptest.NewRecorder()
	checkTimeHandler(rec, httptest.NewRequest(http.MethodGet, "/check-time", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing ts: got %d, want 400", rec.Code)
	}

	// Stale millisecond ts (2s old) -> 400 immediately
	staleMs := time.Now().Add(-2 * time.Second).UnixMilli()
	rec = httptest.NewRecorder()
	checkTimeHandler(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/check-time?ts=%d", staleMs), nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("stale ms ts: got %d, want 400", rec.Code)
	}

	// Fresh millisecond ts -> 200 OK
	freshMs := time.Now().UnixMilli()
	rec = httptest.NewRecorder()
	checkTimeHandler(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/check-time?ts=%d", freshMs), nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("fresh ms ts: got %d, want 200", rec.Code)
	}
}
