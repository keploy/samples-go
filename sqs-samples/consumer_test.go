//go:build !sqs_sdk_v1

package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gin-gonic/gin"
)

// v2Config is the SDK v2 client config of the SQS endpoint at url: no
// retries, so a call the endpoint refuses is the consumer's to handle.
func v2Config(url string) aws.Config {
	return aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider("dummy", "dummy", ""),
		BaseEndpoint: aws.String(url),
		Retryer:      func() aws.Retryer { return aws.NopRetryer{} },
	}
}

// The app is ready only once its first batch is done: the duplicate read,
// the insert and the delete of the message the queue held as it started,
// failed or not. Before then /healthz answers 503, the insert held and the
// delete held alike. Mutation: ready once the insert is done, before the
// delete (at replay the delete would then land in the first test's window).
func TestTheAppIsReadyOnlyOnceItsFirstBatchIsDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name        string
		insertFails bool
		deleteFails bool
	}{{"both done", false, false}, {"the insert failed", true, false}, {"the delete failed", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(forcedVariantEnv, "")
			q := &fakeQueue{deleteFails: tc.deleteFails, deleting: make(chan struct{}, 1), releaseDelete: make(chan struct{})}
			cfg := v2Config(q.serve(t))
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
			if w := getPath(newRouter(rd), "/healthz"); w.Code != http.StatusServiceUnavailable {
				t.Fatalf("with the insert held, /healthz answered %d", w.Code)
			}
			close(release)
			<-q.deleting
			if w := getPath(newRouter(rd), "/healthz"); w.Code != http.StatusServiceUnavailable {
				t.Fatalf("with the delete held, /healthz answered %d", w.Code)
			}
			close(q.releaseDelete)
			<-q.deleted
			<-rd.ready
			if w := getPath(newRouter(rd), "/healthz"); w.Code != http.StatusOK {
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
// of the next delete is told of the next one, not of one before it waited;
// a waiter of the first delete is told of it, whether it came before the
// wait or after. Mutation: forget the first delete once it is past.
func TestTheForcedVariant(t *testing.T) {
	t.Setenv(forcedVariantEnv, "")
	if pollWaitSeconds() != 4 || forcedVariant() {
		t.Fatal("the default build polls long")
	}
	t.Setenv(forcedVariantEnv, "1")
	if pollWaitSeconds() != 0 || !forcedVariant() {
		t.Fatal("the forced variant polls short")
	}
	d := newDeleteSignal()
	first := d.firstEnded()
	select {
	case <-first:
		t.Fatal("a wait for the first delete ended before any delete")
	default:
	}
	d.notify()
	next := d.next()
	select {
	case <-next:
		t.Fatal("a delete before the wait woke it")
	default:
	}
	for _, w := range []<-chan struct{}{first, d.firstEnded()} {
		select {
		case <-w:
		default:
			t.Fatal("the first delete did not end a wait for it, taken before it or after")
		}
	}
	d.notify()
	select {
	case <-next:
	default:
		t.Fatal("the delete after the wait did not wake it")
	}
}

// At replay the consumer can end its work on the message before the window
// test's request reaches the app (its delivery is the test's, served once
// the test opens): the forced variant's wait, taken after that delete, still
// ends. Mutation: wait for the next delete only.
func TestTheForcedWaitEndsOnADeleteThatCameFirst(t *testing.T) {
	forcedWaitTaken.Store(false)
	prev := consumerDeletes
	consumerDeletes = newDeleteSignal()
	t.Cleanup(func() { forcedWaitTaken.Store(false); consumerDeletes = prev })
	t.Setenv(forcedVariantEnv, "1")
	consumerDeletes.notify()
	w := forcedDeleteWait()
	if w == nil {
		t.Fatal("the forced variant's first POST /url did not wait")
	}
	select {
	case <-w:
	default:
		t.Fatal("the consumer's delete came before the wait, and the wait did not end")
	}
}

// Only the forced variant's first parsed POST /url waits for the consumer's
// delete (the window test's: the lane sends one message, for it); every later
// one answers at once, as every POST /url does outside the forced variant.
// Mutation: every parsed POST /url waiting.
func TestOnlyTheForcedVariantsFirstPostWaitsForADelete(t *testing.T) {
	forcedWaitTaken.Store(false)
	t.Cleanup(func() { forcedWaitTaken.Store(false) })
	t.Setenv(forcedVariantEnv, "")
	if forcedDeleteWait() != nil {
		t.Fatal("outside the forced variant a POST /url waited")
	}
	t.Setenv(forcedVariantEnv, "1")
	if forcedDeleteWait() == nil {
		t.Fatal("the forced variant's first POST /url did not wait for a delete")
	}
	if forcedDeleteWait() != nil {
		t.Fatal("a later POST /url of the forced variant waited for a delete")
	}
}

// The forced variant's POST /url waits until the consumer's work on the next
// message ends: its delete answered, or failed (the app logs that it
// failed), or the consumer stopped. Each ends the wait as it comes, never a
// timer. Mutation: tell the waiter of a delete done
// only, or not of the consumer's stop.
func TestTheForcedWaitEndsWithTheMessagesWork(t *testing.T) {
	for _, tc := range []struct {
		name        string
		deleteFails bool
	}{{"the delete done", false}, {"the delete failed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(forcedVariantEnv, "1")
			q := &fakeQueue{deleteFails: tc.deleteFails}
			cfg := v2Config(q.serve(t))
			release := make(chan struct{})
			close(release)
			prev := eventStore
			eventStore = heldEvents{q: q, release: release, read: make(chan struct{}, 1)}
			t.Cleanup(func() { eventStore = prev })
			prevDeletes := consumerDeletes
			consumerDeletes = newDeleteSignal()
			t.Cleanup(func() { consumerDeletes = prevDeletes })
			waiter := consumerDeletes.firstEnded()
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				consumeSQSMessages(ctx, cfg, newReadiness())
			}()
			t.Cleanup(func() {
				cancel()
				<-done
			})
			<-q.deleted
			// The consumer is told the delete's answer at once and ends the
			// wait then: a bound far past that fails a wait that never ends
			// now, not at go test's timeout.
			select {
			case <-waiter:
			case <-time.After(30 * time.Second):
				t.Fatal("the consumer's delete ended, and the forced variant's wait did not")
			}
		})
	}
	t.Run("the consumer stopped", func(t *testing.T) {
		t.Setenv(forcedVariantEnv, "1")
		q := &fakeQueue{}
		cfg := v2Config(q.serve(t))
		prevDeletes := consumerDeletes
		consumerDeletes = newDeleteSignal()
		t.Cleanup(func() { consumerDeletes = prevDeletes })
		waiter := consumerDeletes.firstEnded()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // its first poll fails: it stops
		rd := newReadiness()
		consumeSQSMessages(ctx, cfg, rd)
		if rd.stopErr.Load() == nil {
			t.Fatal("the consumer did not stop")
		}
		select {
		case <-waiter:
		default:
			t.Fatal("the consumer stopped, and a POST /url waiting for its delete was not told")
		}
	})
}
