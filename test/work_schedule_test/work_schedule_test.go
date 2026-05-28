package work_schedule_test

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/evgeniums/evgo/pkg/cache/inmem_cache"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/work_schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

// ---------------------------------------------------------------------------
// Minimal Work type for tests
// ---------------------------------------------------------------------------

type TestWork struct {
	work_schedule.WorkBase
}

func newTestWork() *TestWork { return &TestWork{} }

func dbModels() []interface{} {
	return []interface{}{&TestWork{}}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func initApp(t *testing.T) (app_context.Context, context.Context) {
	return test_utils.InitAppContext(t, testDir, dbModels(), "work_schedule.json")
}

func makeScheduler(t *testing.T, app app_context.Context) *work_schedule.WorkSchedule[*TestWork] {
	return makeSchedulerFull(t, app, nil, nil)
}

func makeSchedulerWithRunner(t *testing.T, app app_context.Context, runner work_schedule.WorkRunner[*TestWork]) *work_schedule.WorkSchedule[*TestWork] {
	return makeSchedulerFull(t, app, runner, nil)
}

// makeSchedulerFull creates a scheduler with an optional runner and an optional
// shared locker. Passing a shared locker simulates multiple pool instances backed
// by a distributed lock (e.g. Redis): both schedulers will contend on the same
// lock namespace, so only one can run a given work at a time.
func makeSchedulerFull(t *testing.T, app app_context.Context, runner work_schedule.WorkRunner[*TestWork], locker cache.Locker) *work_schedule.WorkSchedule[*TestWork] {
	s := work_schedule.NewWorkSchedule[*TestWork]("test_work", work_schedule.Config[*TestWork]{
		WorkBuilder: newTestWork,
		WorkRunner:  runner,
		Locker:      locker,
	})
	require.NoError(t, s.Init(app, "work_scheduler"))
	return s
}

func remaining(t *testing.T, s *work_schedule.WorkSchedule[*TestWork], sctx context.Context) int {
	var works []*TestWork
	_, err := s.CRUD().List(sctx, db.NewFilter(), &works)
	require.NoError(t, err)
	return len(works)
}

// ---------------------------------------------------------------------------
// WorkRunner implementations
// ---------------------------------------------------------------------------

type recordingRunner struct {
	mu   sync.Mutex
	done bool
	runs []string
}

func (r *recordingRunner) Run(_ context.Context, w *TestWork) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs = append(r.runs, w.GetReferenceId())
	return r.done, nil
}

func (r *recordingRunner) calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string{}, r.runs...)
}

type countingRunner struct {
	count atomic.Int32
}

func (r *countingRunner) Run(_ context.Context, w *TestWork) (bool, error) {
	r.count.Add(1)
	return true, nil
}

type typeRecordingRunner struct {
	mu    sync.Mutex
	types map[string]bool
}

func (r *typeRecordingRunner) Run(_ context.Context, w *TestWork) (bool, error) {
	r.mu.Lock()
	if r.types == nil {
		r.types = map[string]bool{}
	}
	r.types[w.GetReferenceType()] = true
	r.mu.Unlock()
	return true, nil
}

func (r *typeRecordingRunner) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []string{}
	for k := range r.types {
		out = append(out, k)
	}
	return out
}

// ---------------------------------------------------------------------------
// Claim / lease behaviour
// ---------------------------------------------------------------------------

func TestDueWorkIsClaimed(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-1", "type-a")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, "ref-1", claimed[0].GetReferenceId())
	assert.Equal(t, "type-a", claimed[0].GetReferenceType())
}

func TestFutureWorkNotClaimed(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-future", "type-b")
	w.SetNextTime(time.Now().Add(10 * time.Minute))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	assert.Empty(t, claimed, "work scheduled in the future must not be claimed")
}

// TestClaimedWorkIsLeased verifies that once a work is claimed it is pushed into
// the future (held), so the same scheduler does not pick it up again within the
// hold window.
func TestClaimedWorkIsLeased(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-hold", "type-c")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	claimed2, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	assert.Empty(t, claimed2, "a claimed work must be leased and not re-claimed within the hold window")
}

// TestConcurrentClaimNoDuplicate exercises the concurrent-claim path: two
// schedulers backed by the SAME database but with INDEPENDENT in-memory lockers
// (simulating two pool instances) both try to claim the same due work at the
// same time. The DB lease + the due-ness re-check under the row lock (the #3
// fix) must ensure the work is handed to at most one of them — never both.
func TestConcurrentClaimNoDuplicate(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	// two independent scheduler instances over the same database
	schedA := makeScheduler(t, app)
	schedB := makeScheduler(t, app)

	_, seedCtx := test_utils.SimpleOpContext(app, "seed")
	const n = 8
	for i := 0; i < n; i++ {
		w := schedA.NewWork(refID(i), "type-race")
		w.SetNextTime(time.Now().Add(-time.Second))
		require.NoError(t, schedA.PostWork(seedCtx, w, work_schedule.SCHEDULE))
	}

	var (
		mu      sync.Mutex
		claimed = map[string]int{}
		wg      sync.WaitGroup
	)
	record := func(works []*TestWork) {
		mu.Lock()
		for _, w := range works {
			claimed[w.GetReferenceId()]++
		}
		mu.Unlock()
	}

	claim := func(s *work_schedule.WorkSchedule[*TestWork]) {
		defer wg.Done()
		_, c := test_utils.SimpleOpContext(app, "claim")
		// tolerate transient sqlite "database is locked" errors under contention;
		// the assertion below only cares that nothing is claimed twice.
		works, _ := s.ClaimDueWorks(c)
		record(works)
	}

	wg.Add(2)
	go claim(schedA)
	go claim(schedB)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	for ref, c := range claimed {
		assert.LessOrEqualf(t, c, 1, "work %s was claimed %d times (must never exceed 1)", ref, c)
	}
}

func TestRemoveWork(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-remove", "type-d")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	require.NoError(t, s.RemoveWork(sctx, "ref-remove", "type-d"))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	assert.Empty(t, claimed, "removed work must not be claimed")
}

func TestDuplicatePostIgnored(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	for i := 0; i < 2; i++ {
		w := s.NewWork("ref-dup", "type-e")
		w.SetNextTime(time.Now().Add(-time.Second))
		require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))
	}

	assert.Equal(t, 1, remaining(t, s, sctx), "posting the same reference twice must not create a second row")
}

// ---------------------------------------------------------------------------
// Processing behaviour (DoWork)
// ---------------------------------------------------------------------------

func TestDoWorkDoneDeletesRow(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &recordingRunner{done: true}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-done", "type-f")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, s.DoWork(sctx, claimed[0]))
	assert.Equal(t, []string{"ref-done"}, runner.calls())
	assert.Equal(t, 0, remaining(t, s, sctx), "a completed (done=true) work must be deleted")
}

func TestDoWorkNotDoneReschedules(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &recordingRunner{done: false}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-retry", "type-g")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, s.DoWork(sctx, claimed[0]))
	require.Len(t, runner.calls(), 1)

	// not done -> rescheduled into the future, so not immediately claimable, but still present
	assert.Equal(t, 1, remaining(t, s, sctx), "a non-done work must remain in the database")
	claimed2, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	assert.Empty(t, claimed2, "a rescheduled work must not be immediately claimable")
}

// ---------------------------------------------------------------------------
// Locking (#1, #2)
// ---------------------------------------------------------------------------

// TestLockerInitialized would panic with a nil-pointer dereference under the old
// inverted nil-check, because AcquireWork would call Lock on a nil locker.
func TestLockerInitialized(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-lock", "type-h")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, s.AcquireWork(sctx, claimed[0]), "AcquireWork must succeed with the default in-memory locker")
	require.NoError(t, s.ReleaseWork(sctx, claimed[0]))
}

func TestAcquireWorkAlreadyLocked(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-lock2", "type-i")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, s.AcquireWork(sctx, claimed[0]))

	// second acquire on the same reference id must report ErrWorkLocked
	err2 := s.AcquireWork(sctx, s.NewWork("ref-lock2", "type-i"))
	assert.ErrorIs(t, err2, work_schedule.ErrWorkLocked)

	require.NoError(t, s.ReleaseWork(sctx, claimed[0]))
}

// TestDoWorkSkipsLockedWork pins the #1 fix: DoWork must honour AcquireWork's
// result and skip a work whose lock is already held, instead of running it.
func TestDoWorkSkipsLockedWork(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &countingRunner{}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-locked", "type-j")
	w.SetNextTime(time.Now().Add(-time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	require.NoError(t, s.AcquireWork(sctx, claimed[0]))

	// DoWork on the same reference must skip silently (no error, runner not called)
	require.NoError(t, s.DoWork(sctx, s.NewWork("ref-locked", "type-j")))
	assert.Equal(t, int32(0), runner.count.Load(), "DoWork must not run a work whose lock is already held")

	require.NoError(t, s.ReleaseWork(sctx, claimed[0]))
}

// ---------------------------------------------------------------------------
// ResetNextTime preserves delay (#5)
// ---------------------------------------------------------------------------

func TestResetNextTimePreservesDelay(t *testing.T) {
	w := &TestWork{}
	w.InitObject()
	w.SetInitialDelay(42)
	w.SetNextTime(time.Now().Add(time.Hour))

	w.ResetNextTime()

	assert.True(t, w.GetNextTime().IsZero(), "ResetNextTime must clear NextTime")
	assert.Equal(t, 42, w.GetInitialDelay(), "ResetNextTime must NOT discard the initial delay")
}

// ---------------------------------------------------------------------------
// End-to-end Run lifecycle (workers + ticker, #4)
// ---------------------------------------------------------------------------

func TestDirectPostRunsInline(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &countingRunner{}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-direct", "type-k")
	require.NoError(t, s.PostWork(sctx, w, work_schedule.DIRECT))

	assert.Equal(t, int32(1), runner.count.Load(), "DIRECT post mode must execute the work inline")
}

func TestQueuedPostProcessedByWorker(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &countingRunner{}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	// Enqueue before Run; the queue is buffered, so this does not block.
	w := s.NewWork("ref-queued", "type-l")
	require.NoError(t, s.PostWork(sctx, w, work_schedule.QUEUED))

	s.Run(context.Background())
	defer s.Shutdown(context.Background())

	require.Eventually(t,
		func() bool { return runner.count.Load() >= 1 },
		6*time.Second, 50*time.Millisecond,
		"QUEUED work must be processed by the worker pool",
	)
}

// TestTickerPicksUpFutureWork pins the #4 fix: a work that becomes due while the
// scheduler is idle (queue drained) must be picked up by the periodic ticker.
func TestTickerPicksUpFutureWork(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &countingRunner{}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	w := s.NewWork("ref-tick", "type-m")
	w.SetNextTime(time.Now().Add(1 * time.Second))
	require.NoError(t, s.PostWork(sctx, w, work_schedule.SCHEDULE))

	s.Run(context.Background())
	defer s.Shutdown(context.Background())

	// not due yet
	time.Sleep(300 * time.Millisecond)
	require.Equal(t, int32(0), runner.count.Load(), "future work must not run before it is due")

	// the periodic ticker (period=1s) must re-scan and pick it up
	require.Eventually(t,
		func() bool { return runner.count.Load() >= 1 },
		6*time.Second, 100*time.Millisecond,
		"periodic ticker must pick up work that became due while idle",
	)
}

func TestRunProcessesMultipleTypes(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &typeRecordingRunner{}
	s := makeSchedulerWithRunner(t, app, runner)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	for _, typ := range []string{"alpha", "beta", "gamma"} {
		w := s.NewWork("ref-"+typ, typ)
		require.NoError(t, s.PostWork(sctx, w, work_schedule.QUEUED))
	}

	s.Run(context.Background())
	defer s.Shutdown(context.Background())

	require.Eventually(t,
		func() bool { return remaining(t, s, sctx) == 0 },
		6*time.Second, 50*time.Millisecond,
		"all queued works must be processed and deleted",
	)
	assert.ElementsMatch(t, []string{"alpha", "beta", "gamma"}, runner.seen())
}

// ---------------------------------------------------------------------------
// Unified dispatch: nudge path via PoolWorkSubscriber
// ---------------------------------------------------------------------------

// TestNudgeDurableWorkClaimedBySubscriber verifies that when a durable (non-no-db)
// QUEUED work is posted, the subscriber's Handle treats the message as a nudge:
// it calls ClaimDueWorks on the local scheduler and enqueues the work, rather
// than running the work payload directly. Two subscriber instances receiving the
// same nudge must together process the work at most once.
func TestNudgeDurableWorkClaimedBySubscriber(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &countingRunner{}

	// Share one locker across both schedulers to simulate a distributed lock
	// (e.g. Redis). Without a shared locker each instance would have its own
	// in-memory lock namespace and could both pass AcquireWork concurrently,
	// which is the correct failure mode in a pool without a distributed locker.
	sharedLocker := inmem_cache.NewLocker()

	// Two independent scheduler instances over the same DB, simulating a pool.
	schedA := makeSchedulerFull(t, app, runner, sharedLocker)
	schedB := makeSchedulerFull(t, app, runner, sharedLocker)

	// Start both worker pools.
	schedA.Run(context.Background())
	defer schedA.Shutdown(context.Background())
	schedB.Run(context.Background())
	defer schedB.Shutdown(context.Background())

	_, sctx := test_utils.SimpleOpContext(app, "post")

	// Post a durable QUEUED work. Under the unified dispatch PostWork persists the
	// row with next_time=now and then the invoker (here the default local dispatcher)
	// would nudge; in this test we drive the nudge manually via HandleMessage.
	w := schedA.NewWork("ref-nudge", "type-nudge")
	require.NoError(t, schedA.PostWork(sctx, w, work_schedule.QUEUED))

	// Build two subscribers backed by the two schedulers.
	subA := work_schedule.NewPoolSubscriber[*TestWork](nil, schedA, "sub-a")
	subB := work_schedule.NewPoolSubscriber[*TestWork](nil, schedB, "sub-b")

	// Deliver the same durable nudge to both subscribers concurrently.
	nudge := &work_schedule.PubsubWork[*TestWork]{NoDb: false}

	var wg sync.WaitGroup
	deliver := func(sub *work_schedule.PoolWorkSubscriber[*TestWork]) {
		defer wg.Done()
		_, c := test_utils.SimpleOpContext(app, "nudge")
		_ = sub.HandleMessage(c, nudge)
	}
	wg.Add(2)
	go deliver(subA)
	go deliver(subB)
	wg.Wait()

	// The work must have been executed exactly once across both subscribers.
	require.Eventually(t,
		func() bool { return runner.count.Load() >= 1 },
		3*time.Second, 50*time.Millisecond,
		"nudge subscriber must claim and process the durable work",
	)
	// Allow a brief settling window, then assert no double execution.
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, int32(1), runner.count.Load(), "durable work must be executed exactly once, not once per nudge recipient")
}

// TestNudgeNoDbWorkRunsDirectly verifies that a no-db work is NOT routed through
// ClaimDueWorks: it is invoked directly from the pubsub payload, so no DB row
// needs to exist.
func TestNudgeNoDbWorkRunsDirectly(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	runner := &countingRunner{}
	sched := makeSchedulerWithRunner(t, app, runner)
	sched.Run(context.Background())
	defer sched.Shutdown(context.Background())

	sub := work_schedule.NewPoolSubscriber[*TestWork](nil, sched, "sub-nodb")

	// Build a no-db PubsubWork — no DB row exists for this reference.
	w := sched.NewWork("ref-nodb-nudge", "type-x")
	w.SetNoDb(true)
	payload := work_schedule.NewPubsubWork(w, work_schedule.QUEUED)
	payload.NoDb = true

	_, sctx := test_utils.SimpleOpContext(app, "nodb-nudge")
	require.NoError(t, sub.HandleMessage(sctx, payload))

	require.Eventually(t,
		func() bool { return runner.count.Load() >= 1 },
		3*time.Second, 50*time.Millisecond,
		"no-db work must be invoked directly from the pubsub payload",
	)
}

// TestPostQueuedDurableNextTimeIsNow verifies that PostWork with QUEUED mode on a
// durable work sets next_time to now (not to a future hold window), so the work is
// immediately claimable by ClaimDueWorks.
func TestPostQueuedDurableNextTimeIsNow(t *testing.T) {
	app, _ := initApp(t)
	defer app.Close()

	s := makeScheduler(t, app)
	_, sctx := test_utils.SimpleOpContext(app, "test")

	before := time.Now()
	w := s.NewWork("ref-now", "type-now")
	require.NoError(t, s.PostWork(sctx, w, work_schedule.QUEUED))
	after := time.Now()

	// The row must be immediately claimable (next_time between before and after).
	claimed, err := s.ClaimDueWorks(sctx)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	nt := claimed[0].GetNextTime()
	assert.False(t, nt.Before(before), "next_time must not be before PostWork was called")
	assert.True(t, nt.After(after), "next_time must be pushed into the future by the claim hold")
	_ = nt
}

func refID(i int) string {
	return "ref-race-" + string(rune('a'+i))
}
