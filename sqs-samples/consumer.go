package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// What both SDKs' consumers share (sqs_v1.go, sqs_v2.go).
//
// The consumer reads a batch, and for each message reads whether its event
// was recorded already (a duplicate read), inserts it into MongoDB, and
// deletes it from the queue. The app is ready (/healthz) once its first
// batch is done, insert and delete included, whether they succeeded or not:
// a message on the queue as the app starts is processed before the app says
// it is ready, so a caller that waits for readiness (keploy test's
// --health-url) finds that work done before its first request. An empty
// first poll makes the app ready at once.
//
// The forced variant (SQS_S12_FORCED=1, keploy's S12 lane) is ready after
// the first poll, before any batch; polls short (WaitTimeSeconds 0, a pause
// after an empty answer), so a message is delivered on a poll sent after it;
// and POST /url, once it has written its URL, waits for the consumer's next
// delete before it answers, so the consumer's work lands inside its request.

// forcedVariantEnv turns the forced variant on.
const forcedVariantEnv = "SQS_S12_FORCED"

// forcedVariant reports whether the forced variant is on.
func forcedVariant() bool {
	on, _ := strconv.ParseBool(os.Getenv(forcedVariantEnv))
	return on
}

// pollWaitSeconds is how long a poll asks SQS to hold it when the queue is
// empty: 4 s, long polls; 0 in the forced variant.
func pollWaitSeconds() int32 {
	if forcedVariant() {
		return 0
	}
	return 4
}

// emptyPollPause is the forced variant's pause after a short poll that found
// nothing, so its polls do not spin.
const emptyPollPause = 200 * time.Millisecond

// afterEmptyPoll pauses after a poll that found nothing, in the forced
// variant only (a long poll held already).
func afterEmptyPoll(ctx context.Context) {
	if !forcedVariant() {
		return
	}
	t := time.NewTimer(emptyPollPause)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// events is where the consumer records each message's event; mongoEvents,
// col2, in the app.
type events interface {
	// recorded is how many times event was recorded already.
	recorded(ctx context.Context, event string) (int64, error)
	insert(ctx context.Context, event string) error
}

var eventStore events = mongoEvents{}

// mongoEvents records events in col2 (keploy.events).
type mongoEvents struct{}

func (mongoEvents) recorded(ctx context.Context, event string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, mongoCallTimeout)
	defer cancel()
	return col2.CountDocuments(ctx, bson.M{"event": event})
}

func (mongoEvents) insert(ctx context.Context, event string) error {
	ctx, cancel := context.WithTimeout(ctx, mongoCallTimeout)
	defer cancel()
	_, err := col2.InsertOne(ctx, bson.D{{Key: "event", Value: event}, {Key: "processed_at", Value: time.Now()}})
	return err
}

// mongoCallTimeout bounds each of the consumer's MongoDB calls: keploy's S12
// lane holds MongoDB (fsyncLock, docker pause) to place the consumer's calls,
// and a hold that outlives the bound fails the call, which the app logs
// ("Failed to read whether ...", "Failed to insert event ..."), so the lane
// fails on that line instead of hanging. A plain deadline: the driver bounds
// the call client-side only, and sends no maxTimeMS (that is client-side
// operation timeouts' doing, which the app does not set), so what keploy
// records of the call does not change with it.
const mongoCallTimeout = 30 * time.Second

// recordEvent records a message's event: it reads first whether the event
// was recorded already (a duplicate read, before the write), then inserts
// it. A failed read does not stop the insert.
func recordEvent(ctx context.Context, event string) error {
	if n, err := eventStore.recorded(ctx, event); err != nil {
		log.Println("Failed to read whether the event was recorded already:", err)
	} else if n > 0 {
		log.Printf("The event was recorded %d time(s) already", n)
	}
	return eventStore.insert(ctx, event)
}

// deleteSignal tells a waiter of the consumer's next delete (the forced
// variant's POST /url).
type deleteSignal struct {
	mu sync.Mutex
	ch chan struct{}
}

// consumerDeletes is the consumer's deletes.
var consumerDeletes = &deleteSignal{ch: make(chan struct{})}

// next is a channel closed at the consumer's next delete after this call.
func (d *deleteSignal) next() <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ch
}

// notify says the consumer deleted a message.
func (d *deleteSignal) notify() {
	d.mu.Lock()
	defer d.mu.Unlock()
	close(d.ch)
	d.ch = make(chan struct{})
}

// forcedWaitMax bounds the forced variant's wait for a delete, under the S12
// lane driver's 30 s timeout, so a lane whose message never comes fails on
// its own wait, not on a request that never ends.
const forcedWaitMax = 25 * time.Second

// waitForDelete waits for the consumer's delete that deleted closes at, or
// until ctx ends, or forcedWaitMax.
func waitForDelete(ctx context.Context, deleted <-chan struct{}) {
	t := time.NewTimer(forcedWaitMax)
	defer t.Stop()
	select {
	case <-deleted:
	case <-ctx.Done():
	case <-t.C:
		log.Println("No delete came while POST /url waited for the consumer (" + forcedVariantEnv + ")")
	}
}
