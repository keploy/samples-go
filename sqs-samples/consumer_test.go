//go:build !sqs_sdk_v1

package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gin-gonic/gin"
)

// fakeQueue is an SQS endpoint holding one message: the first ReceiveMessage
// delivers it, DeleteMessage deletes it (or fails, deleteFails), and every
// later poll waits until its caller gives up or the test ends. It answers in
// the protocol the call is made in (Query, which go.v2query.mod's build
// speaks, or JSON), and notes each call in calls.
type fakeQueue struct {
	mu          sync.Mutex
	calls       []string
	deleteFails bool
	deleted     chan struct{}
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

func (q *fakeQueue) serve(t *testing.T) aws.Config {
	t.Helper()
	q.deleted = make(chan struct{})
	ended := make(chan struct{})
	polls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
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
	return aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
		BaseEndpoint: aws.String(srv.URL),
		Retryer:      func() aws.Retryer { return aws.NopRetryer{} },
	}
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

// The app is ready only once its first batch is done: the duplicate read,
// the insert and the delete of the message the queue held as it started,
// failed or not. Before then /healthz answers 503.
func TestTheAppIsReadyOnlyOnceItsFirstBatchIsDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name        string
		insertFails bool
		deleteFails bool
	}{{"both done", false, false}, {"the insert failed", true, false}, {"the delete failed", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(forcedVariantEnv, "")
			q := &fakeQueue{deleteFails: tc.deleteFails}
			cfg := q.serve(t)
			release := make(chan struct{})
			prev := eventStore
			read := make(chan struct{}, 1)
			eventStore = heldEvents{q: q, release: release, read: read, insertFails: tc.insertFails}
			t.Cleanup(func() { eventStore = prev })
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
			// Each wait is on the event itself (go test's own timeout bounds
			// a test that never sees it): the read, the delete, readiness.
			<-read
			if w := get(newRouter(rd), "/healthz"); w.Code != http.StatusServiceUnavailable {
				t.Fatalf("with the insert held, /healthz answered %d", w.Code)
			}
			close(release)
			<-q.deleted
			<-rd.ready
			if w := get(newRouter(rd), "/healthz"); w.Code != http.StatusOK {
				t.Fatalf("after its first batch, /healthz answered %d", w.Code)
			}
			if got := strings.Join(q.seen()[:4], ","); got != "ReceiveMessage,CountDocuments,InsertOne,DeleteMessage" {
				t.Fatalf("calls %s; want the read before the insert, the delete after it", got)
			}
		})
	}
}

// P, the S12 lane's first test: a POST /url whose body does not parse is
// answered 400 before any MongoDB call (col is nil here: a call would
// panic).
func TestAnUnparseablePostIsRefusedBeforeAnyMongoCall(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var said strings.Builder
	log.SetOutput(&said)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	w := httptest.NewRecorder()
	newRouter(newReadiness()).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/url", strings.NewReader("{not json")))
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "failed to decode req") {
		t.Fatalf("answered %d %q", w.Code, w.Body.String())
	}
	// The lane waits on the handling line for the request it sends after P:
	// P never says it.
	if strings.Contains(said.String(), handlingLine) {
		t.Fatalf("P said %q", handlingLine)
	}
}

// A POST /url it parsed says it handles it before its write: keploy's S12
// lane waits on the line to know the request's window is open, and holds
// MongoDB writes meanwhile, so a line said after the write would never come.
// col is nil here: the write panics, which gin recovers as a 500, after the
// line. Mutation: say it after the write.
func TestAParsedPostSaysItHandlesItBeforeItsWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv(forcedVariantEnv, "")
	var said strings.Builder
	log.SetOutput(&said)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	w := httptest.NewRecorder()
	newRouter(newReadiness()).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/url", strings.NewReader(`{"url": "https://google.com"}`)))
	if !strings.Contains(said.String(), handlingLine) {
		t.Fatalf("answered %d, said %q; want %q before the write", w.Code, said.String(), handlingLine)
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("answered %d; want the write's panic recovered as a 500", w.Code)
	}
}

// The forced variant polls short; the default holds each poll 4 s. A waiter
// of the next delete is told of the next one, not of one before it waited.
func TestTheForcedVariant(t *testing.T) {
	t.Setenv(forcedVariantEnv, "")
	if pollWaitSeconds() != 4 || forcedVariant() {
		t.Fatal("the default build polls long")
	}
	t.Setenv(forcedVariantEnv, "1")
	if pollWaitSeconds() != 0 || !forcedVariant() {
		t.Fatal("the forced variant polls short")
	}
	d := &deleteSignal{ch: make(chan struct{})}
	d.notify()
	next := d.next()
	select {
	case <-next:
		t.Fatal("a delete before the wait woke it")
	default:
	}
	d.notify()
	select {
	case <-next:
	default:
		t.Fatal("the delete after the wait did not wake it")
	}
}
