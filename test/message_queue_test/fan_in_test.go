package message_queue_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/message_queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- FanInBase ---

func TestFanInMergesInputsPreservingPerInputOrder(t *testing.T) {
	fi := message_queue.NewFanIn[int]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fi.Run(ctx)

	chA := make(chan int, 10)
	chB := make(chan int, 10)
	fi.AddInput(chA)
	fi.AddInput(chB)

	for i := 0; i < 5; i++ {
		chA <- i // 0..4
	}
	for i := 0; i < 5; i++ {
		chB <- 100 + i // 100..104
	}

	var sequence []int
	for i := 0; i < 10; i++ {
		select {
		case v := <-fi.Output():
			sequence = append(sequence, v)
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for merged value %d/10", i+1)
		}
	}

	var fromA, fromB []int
	for _, v := range sequence {
		if v < 100 {
			fromA = append(fromA, v)
		} else {
			fromB = append(fromB, v)
		}
	}
	assert.Equal(t, []int{0, 1, 2, 3, 4}, fromA, "values from a single input must arrive in that input's send order")
	assert.Equal(t, []int{100, 101, 102, 103, 104}, fromB)
}

func TestFanInRemoveInputStopsOnlyThatInput(t *testing.T) {
	fi := message_queue.NewFanIn[int]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fi.Run(ctx)

	chA := make(chan int, 1)
	chB := make(chan int, 1)
	fi.AddInput(chA)
	fi.AddInput(chB)

	fi.RemoveInput(chA)

	chA <- 1
	chB <- 2

	select {
	case v := <-fi.Output():
		assert.Equal(t, 2, v, "the remaining input must still be merged")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for value from the remaining input")
	}

	select {
	case v := <-fi.Output():
		t.Fatalf("unexpected value %v from a removed input", v)
	case <-time.After(200 * time.Millisecond):
		// expected: nothing more arrives
	}
}

// TestFanInContextCancelClosesOutputAndDone is the B6 regression: the worker
// goroutine's context-cancellation case used to have an empty body and no
// return, so it busy-spun forever instead of exiting - wg.Wait() (and so
// Close/Done/Output-closing) never completed.
func TestFanInContextCancelClosesOutputAndDone(t *testing.T) {
	fi := message_queue.NewFanIn[int]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fi.Run(ctx)
	fi.AddInput(make(chan int))

	cancel()

	select {
	case <-fi.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("Done() must close after context cancellation")
	}

	_, ok := <-fi.Output()
	assert.False(t, ok, "Output() must be closed once Done() has fired")
}

// TestFanInWorkerDoesNotBusySpinAfterCancel is a second angle on B6: a
// leaked, spinning worker goroutine should show up as a goroutine count that
// never settles back down after cancellation.
func TestFanInWorkerDoesNotBusySpinAfterCancel(t *testing.T) {
	baseline := runtime.NumGoroutine()

	fi := message_queue.NewFanIn[int]()
	ctx, cancel := context.WithCancel(context.Background())
	fi.Run(ctx)
	fi.AddInput(make(chan int))

	cancel()
	<-fi.Done()

	require.Eventually(t, func() bool {
		return runtime.NumGoroutine() <= baseline+2 // small tolerance for runtime/GC housekeeping goroutines
	}, 2*time.Second, 20*time.Millisecond, "worker goroutine(s) must exit after cancellation, not busy-spin forever")
}

// TestFanInAddRemoveInputAfterCloseDoesNotBlock is the B14 regression:
// Add/RemoveInput used to send on unbuffered command channels with no
// shutdown guard, blocking forever once the run loop had exited.
func TestFanInAddRemoveInputAfterCloseDoesNotBlock(t *testing.T) {
	fi := message_queue.NewFanIn[int]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fi.Run(ctx)
	fi.Close()

	done := make(chan struct{})
	go func() {
		fi.AddInput(make(chan int))
		fi.RemoveInput(make(chan int))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("AddInput/RemoveInput after Close must return promptly, not block forever")
	}
}

func TestFanInCloseIdempotentAndSafeWithoutRun(t *testing.T) {
	fi := message_queue.NewFanIn[int]()
	assert.NotPanics(t, fi.Close)
	assert.NotPanics(t, fi.Close, "second Close must not panic")

	_, ok := <-fi.Output()
	assert.False(t, ok)
}

// --- FeederFanInBase ---

func TestFeederFanInMergesFeeders(t *testing.T) {
	ffi := message_queue.NewFeederFanIn[string]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ffi.Run(ctx)

	f1 := message_queue.NewFeeder[string](noopProvider{}, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 4})
	f2 := message_queue.NewFeeder[string](noopProvider{}, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 4})
	ffi.AddFeeder(f1)
	ffi.AddFeeder(f2)

	require.True(t, f1.Push("a"))
	require.True(t, f2.Push("b"))

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case v := <-ffi.Channel():
			got[v.(string)] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for merged value %d/2", i+1)
		}
	}
	assert.True(t, got["a"])
	assert.True(t, got["b"])
}

func TestFeederFanInRemoveFeederStopsForwarding(t *testing.T) {
	ffi := message_queue.NewFeederFanIn[string]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ffi.Run(ctx)

	f1 := message_queue.NewFeeder[string](noopProvider{}, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 4})
	ffi.AddFeeder(f1)
	ffi.RemoveFeeder(f1)

	require.True(t, f1.Push("a"))

	select {
	case v := <-ffi.Channel():
		t.Fatalf("unexpected value %v from a removed feeder", v)
	case <-time.After(200 * time.Millisecond):
	}
}

// TestFeederFanInNextDoesNotBlockAddRemove is the deadlock guard the
// snapshot-then-call discipline in FeederFanInBase.Next buys: Feeder.Next()
// forwards to ConsumerBase.tryNext, which can block, so Next() must never
// hold feedersMu while calling out - otherwise one stuck feeder would stall
// every other Add/Remove/Next call.
func TestFeederFanInNextDoesNotBlockAddRemove(t *testing.T) {
	ffi := message_queue.NewFeederFanIn[string]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ffi.Run(ctx)

	block := make(chan struct{})
	stuck := message_queue.NewFeeder[string](&blockingProvider{block: block}, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 1})
	ffi.AddFeeder(stuck)

	nextDone := make(chan struct{})
	go func() {
		ffi.Next() // calls stuck.Next() -> blocks until "block" is closed
		close(nextDone)
	}()
	time.Sleep(50 * time.Millisecond) // give the goroutine time to actually enter the blocking call

	other := message_queue.NewFeeder[string](noopProvider{}, &message_queue.FeederConfig{FEEDER_CHANNEL_DEPTH: 1})
	addRemoveDone := make(chan struct{})
	go func() {
		ffi.AddFeeder(other)
		ffi.RemoveFeeder(other)
		close(addRemoveDone)
	}()

	select {
	case <-addRemoveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("AddFeeder/RemoveFeeder must not block behind a stuck Feeder.Next() call")
	}

	close(block) // unblock the stuck Next() call so its goroutine can exit
	select {
	case <-nextDone:
	case <-time.After(2 * time.Second):
		t.Fatal("stuck Next() call never returned after being unblocked")
	}
}

// --- SubscriberFanInBase ---

func TestSubscriberFanInMergesAndForwardsNext(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfi.Run(ctx)

	s1 := newFakeSubscriber(4)
	s2 := newFakeSubscriber(4)
	sfi.AddSubscriber(s1)
	sfi.AddSubscriber(s2)

	s1.push("a")
	s2.push("b")

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case v := <-sfi.Channel():
			got[v.(string)] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for merged value %d/2", i+1)
		}
	}
	assert.True(t, got["a"])
	assert.True(t, got["b"])

	sfi.Next()
	require.Eventually(t, func() bool {
		return s1.nextCallCount() >= 1 && s2.nextCallCount() >= 1
	}, time.Second, 10*time.Millisecond, "Next() must be forwarded to every current subscriber")
}

func TestSubscriberFanInRemoveSubscriberUnsubscribesIt(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfi.Run(ctx)

	s1 := newFakeSubscriber(1)
	sfi.AddSubscriber(s1)
	sfi.RemoveSubscriber(s1)

	assert.True(t, s1.wasUnsubscribed())
}

// TestSubscriberFanInUnsubscribeUnsubscribesAllExactlyOnce is the B1
// end-to-end-adjacent regression for this layer: Unsubscribe is
// sync.Once-guarded and must release every subscriber exactly once even
// when called concurrently from several goroutines.
func TestSubscriberFanInUnsubscribeUnsubscribesAllExactlyOnce(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfi.Run(ctx)

	s1 := newFakeSubscriber(1)
	s2 := newFakeSubscriber(1)
	sfi.AddSubscriber(s1)
	sfi.AddSubscriber(s2)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sfi.Unsubscribe(context.Background())
		}()
	}
	wg.Wait()

	assert.True(t, s1.wasUnsubscribed())
	assert.True(t, s2.wasUnsubscribed())
	assert.Equal(t, 1, s1.unsubscribeCount())
	assert.Equal(t, 1, s2.unsubscribeCount())

	_, ok := <-sfi.Channel()
	assert.False(t, ok, "Channel() must be closed after Unsubscribe")
}

// TestSubscriberFanInAddSubscriberAfterCloseUnsubscribesInstead is the B14
// regression for this layer: adding a subscriber after shutdown used to
// either block forever or silently leak the subscriber's own subscription.
func TestSubscriberFanInAddSubscriberAfterCloseUnsubscribesInstead(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfi.Run(ctx)
	sfi.Unsubscribe(context.Background())

	late := newFakeSubscriber(1)
	done := make(chan struct{})
	go func() {
		sfi.AddSubscriber(late)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("AddSubscriber after shutdown must not block")
	}
	assert.True(t, late.wasUnsubscribed(), "a subscriber added after shutdown must be unsubscribed, not leaked")
}

func TestSubscriberFanInUnsubscribeSafeWithoutRun(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	assert.NotPanics(t, func() { sfi.Unsubscribe(context.Background()) })
}

// TestSubscriberFanInContextCancellationUnsubscribesAll checks that external
// cancellation (not just an explicit Unsubscribe call) still releases every
// subscription, as it did before the fan-in refactor.
func TestSubscriberFanInContextCancellationUnsubscribesAll(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfi.Run(ctx)

	s1 := newFakeSubscriber(1)
	sfi.AddSubscriber(s1)

	cancel()

	require.Eventually(t, func() bool {
		return s1.wasUnsubscribed()
	}, 2*time.Second, 20*time.Millisecond, "external context cancellation must still release every subscription")
}

// TestSubscriberFanInNextDoesNotBlockAddRemoveSubscriber mirrors the
// FeederFanIn deadlock guard: a subscriber whose Next() blocks must not
// stall AddSubscriber/RemoveSubscriber on another goroutine.
func TestSubscriberFanInNextDoesNotBlockAddRemoveSubscriber(t *testing.T) {
	sfi := message_queue.NewSubscriberFanIn[string, testMessage]()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sfi.Run(ctx)

	block := make(chan struct{})
	stuck := newFakeSubscriber(1)
	stuck.blockNext = block
	sfi.AddSubscriber(stuck)

	nextDone := make(chan struct{})
	go func() {
		sfi.Next()
		close(nextDone)
	}()
	time.Sleep(50 * time.Millisecond)

	other := newFakeSubscriber(1)
	addRemoveDone := make(chan struct{})
	go func() {
		sfi.AddSubscriber(other)
		sfi.RemoveSubscriber(other)
		close(addRemoveDone)
	}()

	select {
	case <-addRemoveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("AddSubscriber/RemoveSubscriber must not block behind a stuck Next() call")
	}

	close(block) // unblock the stuck Next() call so its goroutine can exit
	select {
	case <-nextDone:
	case <-time.After(2 * time.Second):
		t.Fatal("stuck Next() call never returned after being unblocked")
	}
}
