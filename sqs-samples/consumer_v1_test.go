//go:build sqs_sdk_v1

package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/gin-gonic/gin"
)

// v1Client is an SDK v1 (Query protocol) client of the SQS endpoint at url:
// no retries, so a call the endpoint refuses is the consumer's to handle.
func v1Client(t *testing.T, url string) *sqs.SQS {
	t.Helper()
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Endpoint:    aws.String(url),
		Credentials: credentials.NewStaticCredentials("dummy", "dummy", ""),
		MaxRetries:  aws.Int(0),
	})
	if err != nil {
		t.Fatal(err)
	}
	return sqs.New(sess)
}

// The v1 consumer (sqs_v1.go), as the v2 one: the app is ready only once its
// first batch is done, the duplicate read, the insert and the delete, failed
// or not. Mutation: ready before the delete.
func TestTheV1AppIsReadyOnlyOnceItsFirstBatchIsDone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name        string
		insertFails bool
		deleteFails bool
	}{{"both done", false, false}, {"the insert failed", true, false}, {"the delete failed", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(forcedVariantEnv, "")
			q := &fakeQueue{deleteFails: tc.deleteFails, deleting: make(chan struct{}, 1), releaseDelete: make(chan struct{})}
			client := v1Client(t, q.serve(t))
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
				consumeSQSMessages(ctx, client, rd)
			}()
			t.Cleanup(func() {
				cancel()
				<-done
			})
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

// The v1 consumer, as the v2 one, ends the forced variant's wait when its
// work on the next message ends, the delete done or failed, and when it
// stops. Mutation: tell the waiter of a delete done only.
func TestTheV1ForcedWaitEndsWithTheMessagesWork(t *testing.T) {
	for _, tc := range []struct {
		name        string
		deleteFails bool
	}{{"the delete done", false}, {"the delete failed", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(forcedVariantEnv, "1")
			q := &fakeQueue{deleteFails: tc.deleteFails}
			client := v1Client(t, q.serve(t))
			release := make(chan struct{})
			close(release)
			prev := eventStore
			eventStore = heldEvents{q: q, release: release, read: make(chan struct{}, 1)}
			t.Cleanup(func() { eventStore = prev })
			waiter := consumerDeletes.next()
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			go func() {
				defer close(done)
				consumeSQSMessages(ctx, client, newReadiness())
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
		client := v1Client(t, q.serve(t))
		waiter := consumerDeletes.next()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		rd := newReadiness()
		consumeSQSMessages(ctx, client, rd)
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
