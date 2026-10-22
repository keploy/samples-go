package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
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
// and the first POST /url it parses, once it has written its URL, waits for
// the consumer's next delete to end (done or failed, or the consumer's stop)
// before it answers, so the consumer's work lands inside that request. Every later POST /url answers at once: the lane sends
// one message, for that request.

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

// deleteSignal tells a waiter (the forced variant's POST /url) that the
// consumer's work on its next message is over: its delete was answered, or
// failed. It tells it too when the consumer stops: no message's work is
// coming then.
type deleteSignal struct {
	mu sync.Mutex
	ch chan struct{}
}

// consumerDeletes is the consumer's deletes.
var consumerDeletes = &deleteSignal{ch: make(chan struct{})}

// next is a channel closed at the end of the consumer's next delete after
// this call, done or failed, or at its stop.
func (d *deleteSignal) next() <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ch
}

// notify says the consumer's delete of a message ended, or it stopped.
func (d *deleteSignal) notify() {
	d.mu.Lock()
	defer d.mu.Unlock()
	close(d.ch)
	d.ch = make(chan struct{})
}

// forcedWaitTaken is whether a POST /url took the forced variant's wait.
var forcedWaitTaken atomic.Bool

// forcedDeleteWait is what a parsed POST /url waits on once it has written
// its URL: in the forced variant, for the first one the app parses, the
// consumer's next delete (consumerDeletes.next, taken now, before the write);
// nil, no wait, for every later one, and outside the forced variant.
func forcedDeleteWait() <-chan struct{} {
	if !forcedVariant() || !forcedWaitTaken.CompareAndSwap(false, true) {
		return nil
	}
	return consumerDeletes.next()
}

// forcedWaitMax ends the forced variant's wait for the consumer's delete when
// nothing does: neither the delete, done or failed, nor the consumer's stop,
// which end it as they come (deleteSignal). A message's work ends on its own
// (its two MongoDB calls at mongoCallTimeout, 30 s, each), so a wait longer
// than that work can take means no message came. The app then says so
// (noDelete), and keploy's S12 lane fails on that line: a hold that outlives
// the wait fails the lane with a message, never a recording it judges.
const forcedWaitMax = 3 * mongoCallTimeout

// noDelete is what the app says when the forced variant's wait ran out.
const noDelete = "No message's work ended while POST /url waited for the consumer (" + forcedVariantEnv + ")"

// waitForDelete waits until the consumer's delete that deleted closes at
// ends, done or failed, or the consumer stops, or ctx ends, or forcedWaitMax.
func waitForDelete(ctx context.Context, deleted <-chan struct{}) {
	t := time.NewTimer(forcedWaitMax)
	defer t.Stop()
	select {
	case <-deleted:
	case <-ctx.Done():
	case <-t.C:
		log.Println(noDelete)
	}
}
