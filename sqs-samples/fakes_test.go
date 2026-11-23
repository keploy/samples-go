package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// What the tests of both SDKs' consumers share (consumer_test.go,
// consumer_v1_test.go): an SQS endpoint, an event store, a GET.

// fakeQueue is an SQS endpoint holding one message: the first ReceiveMessage
// delivers it, DeleteMessage deletes it (or fails, deleteFails), and every
// later poll waits until its caller gives up or the test ends. It answers in
// the protocol the call is made in (Query, which go.v2query.mod's build
// speaks, or JSON), and notes each call in calls. With deleting set, a
// DeleteMessage tells deleting as it arrives, and is answered only once
// releaseDelete is closed.
type fakeQueue struct {
	mu            sync.Mutex
	calls         []string
	deleteFails   bool
	deleted       chan struct{}
	deleting      chan struct{}
	releaseDelete chan struct{}
}

func (q *fakeQueue) note(c string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.calls = append(q.calls, c)
}

func (q *fakeQueue) seen() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]string(nil), q.calls...)
}

func (q *fakeQueue) serve(t *testing.T) string {
	t.Helper()
	q.deleted = make(chan struct{})
	ended := make(chan struct{})
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			// The caller gave up before it sent its request whole: the
			// consumer's next poll, cancelled as the test ends. No call.
			return
		}
		query := r.Header.Get("X-Amz-Target") == ""
		op := strings.TrimPrefix(r.Header.Get("X-Amz-Target"), "AmazonSQS.")
		if query {
			form, _ := url.ParseQuery(string(body))
			op = form.Get("Action")
		}
		q.note(op)
		if query {
			w.Header().Set("Content-Type", "text/xml")
		} else {
			w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		}
		switch op {
		case "ReceiveMessage":
			q.mu.Lock()
			polls++
			n := polls
			q.mu.Unlock()
			if n > 1 {
				select {
				case <-r.Context().Done():
				case <-ended:
				}
				return
			}
			if query {
				_, _ = w.Write([]byte(`<ReceiveMessageResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/"><ReceiveMessageResult><Message><MessageId>m-1</MessageId><ReceiptHandle>r-1</ReceiptHandle><MD5OfBody>c4ca4238a0b923820dcc509a6f75849b</MD5OfBody><Body>1</Body></Message></ReceiveMessageResult></ReceiveMessageResponse>`))
				return
			}
			_, _ = w.Write([]byte(`{"Messages":[{"MessageId":"m-1","ReceiptHandle":"r-1","MD5OfBody":"c4ca4238a0b923820dcc509a6f75849b","Body":"1"}]}`))
		case "DeleteMessage":
			defer close(q.deleted)
			if q.deleting != nil {
				q.deleting <- struct{}{}
				select {
				case <-q.releaseDelete:
				case <-r.Context().Done():
				case <-ended:
				}
			}
			if q.deleteFails {
				// Headers before the status: what follows WriteHeader is not sent.
				if !query {
					w.Header().Set("x-amzn-query-error", "ReceiptHandleIsInvalid;Sender")
				}
				w.WriteHeader(http.StatusBadRequest)
				if query {
					_, _ = w.Write([]byte(`<ErrorResponse><Error><Type>Sender</Type><Code>ReceiptHandleIsInvalid</Code><Message>no</Message></Error><RequestId>r</RequestId></ErrorResponse>`))
					return
				}
				_, _ = w.Write([]byte(`{"__type":"com.amazonaws.sqs#ReceiptHandleIsInvalid","message":"no"}`))
				return
			}
			if query {
				_, _ = w.Write([]byte(`<DeleteMessageResponse xmlns="http://queue.amazonaws.com/doc/2012-11-05/"><ResponseMetadata><RequestId>r</RequestId></ResponseMetadata></DeleteMessageResponse>`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Errorf("the consumer called %q", op)
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(ended) })
	return srv.URL
}

// heldEvents is an event store whose insert waits for release, noting each
// call, and telling read of each read; insertFails makes the insert fail.
type heldEvents struct {
	q           *fakeQueue
	release     chan struct{}
	read        chan struct{}
	insertFails bool
}

func (e heldEvents) recorded(context.Context, string) (int64, error) {
	e.q.note("CountDocuments")
	select {
	case e.read <- struct{}{}:
	default:
	}
	return 0, nil
}

func (e heldEvents) insert(context.Context, string) error {
	<-e.release
	e.q.note("InsertOne")
	if e.insertFails {
		return errors.New("insert refused")
	}
	return nil
}

// get answers a GET of path from r.
func getPath(r http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

// afterEmptyPoll pauses only in the forced variant, whose polls are short,
// and there until its pause is over or ctx ends; a long poll held already.
// Mutation: no pause in the forced variant, or a pause outside it.
func TestAfterEmptyPollPausesOnlyTheForcedVariant(t *testing.T) {
	t.Setenv(forcedVariantEnv, "")
	start := time.Now()
	afterEmptyPoll(context.Background())
	if d := time.Since(start); d >= emptyPollPause {
		t.Fatalf("the default build paused %v after an empty long poll", d)
	}
	t.Setenv(forcedVariantEnv, "1")
	start = time.Now()
	afterEmptyPoll(context.Background())
	if d := time.Since(start); d < emptyPollPause {
		t.Fatalf("the forced variant paused %v after an empty short poll; want %v", d, emptyPollPause)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start = time.Now()
	afterEmptyPoll(ctx)
	if d := time.Since(start); d >= emptyPollPause {
		t.Fatalf("the forced variant's pause outlived its context by %v", d)
	}
}
