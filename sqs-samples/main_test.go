//go:build !sqs_sdk_v1

package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gin-gonic/gin"
)

func get(r http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// fakeSQS is an SQS endpoint whose first ReceiveMessage gets first's answer,
// and every later one waits until its caller gives up or the test ends.
// again closes when the second one comes: the consumer is past its first
// poll. It answers in the protocol the call is made in (query: SQS's Query
// protocol, which go.v2query.mod's build speaks; else JSON).
func fakeSQS(t *testing.T, first func(w http.ResponseWriter, query bool)) (cfg aws.Config, again <-chan struct{}) {
	t.Helper()
	var calls atomic.Int32
	ended := make(chan struct{})
	second := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read whole, so the server notices the caller hanging up.
		body, _ := io.ReadAll(r.Body)
		query := r.Header.Get("X-Amz-Target") == ""
		if query {
			if form, err := url.ParseQuery(string(body)); err != nil || form.Get("Action") != "ReceiveMessage" {
				t.Errorf("the consumer called %q", body)
			}
		} else if got := r.Header.Get("X-Amz-Target"); got != "AmazonSQS.ReceiveMessage" {
			t.Errorf("the consumer called %q", got)
		}
		if n := calls.Add(1); n > 1 {
			if n == 2 {
				close(second)
			}
			select {
			case <-r.Context().Done():
			case <-ended:
			}
			return
		}
		if query {
			w.Header().Set("Content-Type", "text/xml")
		} else {
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		}
		first(w, query)
	}))
	// Cleanups run last first: the waiting polls end before Close waits on
	// them.
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(ended) })
	return aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
		BaseEndpoint: aws.String(srv.URL),
		Retryer:      func() aws.Retryer { return aws.NopRetryer{} },
	}, second
}

// /healthz is 503 until the SQS consumer's first poll came back, 200 after,
// and the URL shortener's routes are still there.
func TestHealthzWaitsForTheFirstSQSPoll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rd := newReadiness()
	r := newRouter(rd)
	if w := get(r, "/healthz"); w.Code != http.StatusServiceUnavailable || w.Body.String() != "waiting for the first SQS poll" {
		t.Fatalf("before the first poll /healthz answered %d %q", w.Code, w.Body.String())
	}
	rd.markPolled()
	if w := get(r, "/healthz"); w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("after the first poll /healthz answered %d %q", w.Code, w.Body.String())
	}
	routes := map[string]bool{}
	for _, rt := range r.Routes() {
		routes[rt.Method+" "+rt.Path] = true
	}
	for _, want := range []string{"GET /:param", "POST /url"} {
		if !routes[want] {
			t.Fatalf("the router has no %s: %v", want, routes)
		}
	}
}

// The consumer marks the app ready once its first ReceiveMessage comes back
// with no messages: by its second poll, /healthz answers 200, and it still
// does once the consumer has stopped.
func TestTheConsumersFirstPollMakesTheAppReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg, again := fakeSQS(t, func(w http.ResponseWriter, query bool) {
		if query {
			_, _ = w.Write([]byte(`<ReceiveMessageResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/"><ReceiveMessageResult/></ReceiveMessageResponse>`))
			return
		}
		_, _ = w.Write([]byte(`{"Messages":[]}`))
	})
	rd := newReadiness()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		consumeSQSMessages(ctx, cfg, rd)
	}()
	// Before the fake endpoint closes (cleanups run last first): its waiting
	// poll ends when the consumer gives up.
	t.Cleanup(func() {
		cancel()
		<-done
	})
	select {
	case <-again:
	case <-done:
		t.Fatal("the consumer stopped before it polled again")
	}
	if w := get(newRouter(rd), "/healthz"); w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("after the first poll /healthz answered %d %q", w.Code, w.Body.String())
	}
	cancel()
	<-done
	if w := get(newRouter(rd), "/healthz"); w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("once the consumer stopped after its first poll, /healthz answered %d %q", w.Code, w.Body.String())
	}
}

// A consumer that stops before its first poll came back leaves the app
// unready, and /healthz says why.
func TestHealthzSaysWhyTheConsumerStopped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg, again := fakeSQS(t, func(w http.ResponseWriter, query bool) {
		if query {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`<ErrorResponse><Error><Type>Sender</Type><Code>AWS.SimpleQueueService.NonExistentQueue</Code><Message>The specified queue does not exist.</Message></Error><RequestId>r-1</RequestId></ErrorResponse>`))
			return
		}
		w.Header().Set("x-amzn-query-error", "AWS.SimpleQueueService.NonExistentQueue;Sender")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"__type":"com.amazonaws.sqs#QueueDoesNotExist","message":"The specified queue does not exist."}`))
	})
	rd := newReadiness()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		consumeSQSMessages(ctx, cfg, rd)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	select {
	case <-done:
	case <-again:
		t.Fatal("the consumer polled again after its poll failed")
	}
	w := get(newRouter(rd), "/healthz")
	if w.Code != http.StatusServiceUnavailable || !strings.HasPrefix(w.Body.String(), "SQS consumer stopped: ") || !strings.Contains(w.Body.String(), "does not exist") || strings.Contains(w.Body.String(), "waiting") {
		t.Fatalf("after the consumer stopped /healthz answered %d %q", w.Code, w.Body.String())
	}
}
