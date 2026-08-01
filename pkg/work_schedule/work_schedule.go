package work_schedule

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/background_worker"
	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/evgeniums/evgo/pkg/cache/inmem_cache"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type PostMode int

const (
	SCHEDULE PostMode = 0
	DIRECT   PostMode = 1
	QUEUED   PostMode = 2
)

func Mode(m string) PostMode {
	switch m {
	case "schedule":
		return SCHEDULE
	case "direct":
		return DIRECT
	case "queued":
		return QUEUED
	}
	return SCHEDULE
}

// ErrWorkLocked is returned by AcquireWork when the work is already being
// processed elsewhere. It is an expected condition, not a failure, so callers
// should skip the work silently rather than logging it as an error.
var ErrWorkLocked = errors.New("work already locked")

type Work interface {
	common.Object

	GetReferenceType() string
	SetReferenceType(string)

	GetReferenceId() string
	SetReferenceId(string)

	GetNextTime() time.Time
	SetNextTime(time.Time)
	ResetNextTime()

	GetInitialDelay() int
	SetInitialDelay(int)

	SetLock(cache.Lock)
	GetLock() cache.Lock

	SetNoDb(enable bool)
	IsNoDb() bool
}

type WorkBuilder[T Work] func() T

type WorkInvoker[T Work] func(sctx context.Context, work T, postMode PostMode, tenancy ...multitenancy.Tenancy) error

type WorkScheduler[T Work] interface {
	NewWork(referenceId string, referenceType string) T
	AcquireWork(sctx context.Context, work T) error
	ReleaseWork(sctx context.Context, work T) error
	PostWork(sctx context.Context, work T, postMode PostMode, tenancy ...multitenancy.Tenancy) error
	RemoveWork(sctx context.Context, referenceId string, referenceType string) error
}

type WorkSchedulerBase[T Work] struct {
	workBuilder WorkBuilder[T]
}

func (s *WorkSchedulerBase[T]) Construct(workBuilder WorkBuilder[T]) {
	s.workBuilder = workBuilder
}

func (s *WorkSchedulerBase[T]) NewWork(referenceId string, referenceType string) T {
	w := s.workBuilder()
	w.InitObject()
	w.SetReferenceId(referenceId)
	w.SetReferenceType(referenceType)
	return w
}

func (s *WorkSchedulerBase[T]) AcquireWork(sctx context.Context, work T) error {
	return nil
}

func (s *WorkSchedulerBase[T]) ReleaseWork(sctx context.Context, work T) error {
	return nil
}

func (s *WorkSchedulerBase[T]) PostWork(sctx context.Context, work T, postMode PostMode, tenancy ...multitenancy.Tenancy) error {
	return nil
}

func (s *WorkSchedulerBase[T]) RemoveWork(sctx context.Context, referenceId string) error {
	return nil
}

type WorkRunner[T Work] interface {
	Run(sctx context.Context, work T) (bool, error)
}

type WorkBase struct {
	common.ObjectBase
	ReferenceId   string    `json:"reference_id" gorm:"index;index:,unique,composite:ref"`
	ReferenceType string    `json:"reference_type" gorm:"index;index:,unique,composite:ref"`
	NextTime      time.Time `json:"next_time" gorm:"index"`
	NextTimeSet   bool      `json:"next_time_set" gorm:"index;default:false"`

	lock  cache.Lock `json:"-" gorm:"-:all"`
	initialDelay int `json:"-" gorm:"-:all"`
	noDb  bool       `json:"-" gorm:"-:all"`
}

func (w *WorkBase) GetReferenceType() string {
	return w.ReferenceType
}

func (w *WorkBase) SetReferenceType(referenceType string) {
	w.ReferenceType = referenceType
}

func (w *WorkBase) GetReferenceId() string {
	return w.ReferenceId
}

func (w *WorkBase) SetReferenceId(referenceId string) {
	w.ReferenceId = referenceId
}

func (w *WorkBase) GetNextTime() time.Time {
	return w.NextTime
}

func (w *WorkBase) SetNextTime(nextTime time.Time) {
	w.NextTime = nextTime
}

func (w *WorkBase) ResetNextTime() {
	w.SetNextTime(time.Time{})
}

func (w *WorkBase) GetLock() cache.Lock {
	return w.lock
}

func (w *WorkBase) SetLock(lock cache.Lock) {
	w.lock = lock
}

func (w *WorkBase) GetInitialDelay() int {
	return w.initialDelay
}

func (w *WorkBase) SetInitialDelay(seconds int) {
	w.initialDelay = seconds
}

func (w *WorkBase) SetNoDb(enable bool) {
	w.noDb = enable
}

func (w *WorkBase) IsNoDb() bool {
	return w.noDb
}

type WorkInTenancyBase struct {
	WorkBase
	TenancyId string `json:"tenancy_id" gorm:"index"`
}

func (w *WorkInTenancyBase) SetTenancy(tenancy multitenancy.Tenancy) {
	w.TenancyId = tenancy.GetID()
}

func (w *WorkInTenancyBase) GetTenancyId() string {
	return w.TenancyId
}

type WorkScheduleConfig struct {
	PARALLEL_JOBS               int `default:"8"`
	BUCKET_SIZE                 int `default:"32"`
	INVOKATION_INTERVAL_SECONDS int `default:"300"`
	HOLD_WORK_SECONDS           int `default:"900"`
	LOCK_TTL_SECONDS            int `default:"900"`
	PERIOD                      int `default:"5"`
	LOG_EMPTY_WORKS             bool
}

type workItem[T Work] struct {
	work    T
	tenancy multitenancy.Tenancy
}

type WorkSchedule[T Work] struct {
	WorkScheduleConfig
	app_context.WithAppBase
	crud.WithCRUDBase
	background_worker.JobRunnerBase

	WorkSchedulerBase[T]

	name string

	runningWorkCount atomic.Int32
	workQueueSize    atomic.Int32

	workRunner WorkRunner[T]

	queue chan workItem[T]

	invoker WorkInvoker[T]

	locker cache.Locker
	db     db.DB

	wg sync.WaitGroup

	readNextWorks chan struct{}
	stop          chan struct{}
}

type Config[T Work] struct {
	WorkBuilder WorkBuilder[T]
	WorkRunner  WorkRunner[T]
	WorkInvoker WorkInvoker[T]
	Locker      cache.Locker
}

func NewWorkSchedule[T Work](name string, config Config[T], cruds ...crud.CRUD) *WorkSchedule[T] {
	s := &WorkSchedule[T]{
		name:       name,
		workRunner: config.WorkRunner,
	}
	s.WorkSchedulerBase.Construct(config.WorkBuilder)
	s.WithCRUDBase.Construct(cruds...)
	s.invoker = config.WorkInvoker
	if s.invoker == nil {
		s.invoker = s.InvokeWork
	}

	s.locker = config.Locker
	if s.locker == nil {
		s.locker = inmem_cache.NewLocker()
	}

	return s
}

func (s *WorkSchedule[T]) Config() interface{} {
	return &s.WorkScheduleConfig
}

func (s *WorkSchedule[T]) Init(app app_context.Context, configPath ...string) error {

	s.WithAppBase.Init(app)

	err := object_config.LoadLogValidateApp(app, s, "work_schedule", configPath...)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of WorkSchedule", err)
	}

	// create channels
	s.queue = make(chan workItem[T], s.BUCKET_SIZE)
	s.readNextWorks = make(chan struct{}, 1)
	s.stop = make(chan struct{})

	// done
	return nil
}

func (s *WorkSchedule[T]) Shutdown(ctx context.Context) {
	close(s.stop)
	s.wg.Wait()
}

func (s *WorkSchedule[T]) SetRunner(runner WorkRunner[T]) {
	s.workRunner = runner
}

// Wake nudges the scheduler to rescan the DB for due works immediately,
// instead of waiting for the next PERIOD tick. Non-blocking and idempotent:
// if the scheduler is sleeping it wakes it; if a scan is already pending or
// in progress it is a no-op. Safe to call before Run / after Shutdown (the
// select default fires) and inside a DB transaction (it only triggers a
// rescan of committed rows — it never touches the in-memory work object).
func (s *WorkSchedule[T]) Wake() {
	select {
	case s.readNextWorks <- struct{}{}:
	default:
	}
}

func (s *WorkSchedule[T]) AcquireWork(sctx context.Context, work T) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.AcquireWork")
	defer ctx.TraceOutMethod()

	// lock work in cache
	lock, err := cache.LockObject(s.locker, "work_lock", work.GetReferenceId(), s.LOCK_TTL_SECONDS)
	if err != nil {
		c.SetLoggerField("work_reference_id", work.GetReferenceId())
		c.SetMessage("failed to lock work")
		return c.SetError(err)
	}
	if lock == nil {
		c.SetLoggerField("work_reference_id", work.GetReferenceId())
		return ErrWorkLocked
	}
	s.runningWorkCount.Add(1)
	work.SetLock(lock)

	releaseLock := func() {
		s.runningWorkCount.Add(-1)
		lock.Release()
		work.SetLock(nil)
	}

	// For durable works, verify the row still exists under the lock. Another pool
	// instance may have claimed, processed, and deleted the same row between
	// ClaimDueWorks and this point. The cache lock guarantees only one worker
	// proceeds past here for a given referenceId.
	if !work.IsNoDb() {
		probe := s.workBuilder()
		found, err := s.CRUD().Read(sctx, db.Fields{"id": work.GetID()}, probe)
		if err != nil {
			releaseLock()
			c.SetMessage("failed to check work existence")
			return c.SetError(err)
		}
		if !found {
			releaseLock()
			c.SetLoggerField("work_reference_id", work.GetReferenceId())
			return ErrWorkLocked
		}
	}

	// reset next time flag
	err = s.CRUD().Update(sctx, work, db.Fields{"next_time_set": false})
	if err != nil {
		releaseLock()
		c.SetMessage("failed to save next work time set flag in database")
		return c.SetError(err)
	}

	// done
	return nil
}

func (s *WorkSchedule[T]) ReleaseWork(sctx context.Context, work T) error {

	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.ReleaseWork")
	defer ctx.TraceOutMethod()

	lock := work.GetLock()
	if lock != nil {
		s.runningWorkCount.Add(-1)
		err = lock.Release()
		if err != nil {
			c.SetLoggerField("work_reference_id", work.GetReferenceId())
			c.SetMessage("failed to release work")
			return c.SetError(err)
		}
	}

	return nil
}

func (s *WorkSchedule[T]) SetNextWorkTime(work T, reset ...bool) {
	defaultNextTime := utils.OptionalArg(false, reset...)
	if defaultNextTime || work.GetInitialDelay() == 0 && work.GetNextTime().IsZero() {
		work.SetNextTime(time.Now().Add(time.Second * time.Duration(s.INVOKATION_INTERVAL_SECONDS)))
	} else if work.GetNextTime().IsZero() && work.GetInitialDelay() != 0 {
		work.SetNextTime(time.Now().Add(time.Second * time.Duration(work.GetInitialDelay())))
	}
}

func (s *WorkSchedule[T]) PostWork(sctx context.Context, work T, postMode PostMode, tenancy ...multitenancy.Tenancy) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.PostWork")
	defer ctx.TraceOutMethod()

	// set next time
	if postMode == SCHEDULE {
		s.SetNextWorkTime(work)
	} else if !work.IsNoDb() {
		// immediately due so the nudge subscriber can claim it right away;
		// the hold lease is applied atomically inside ClaimDueWorks
		work.SetNextTime(time.Now())
	}

	if work.IsNoDb() {
		// no-db: no DB row; publish the work payload directly to the invoker
		err = s.invoker(sctx, work, postMode, tenancy...)
		if err != nil {
			c.SetMessage("failed to invoke work")
			return err
		}
		return nil
	}

	// Works built directly as struct literals (not via NewWork) may have an empty
	// primary key. CRUD.Update later keys the WHERE clause off obj.GetID(); if the
	// id is "" GORM emits an UPDATE with no WHERE and returns ErrMissingWhereClause.
	// InitObject is idempotent for already-initialised works (it regenerates id and
	// timestamps, but the unique constraint on reference_id/reference_type prevents
	// duplicate durable rows regardless). When the id is already set, skip to avoid
	// altering created_at on a work that was constructed intentionally via NewWork.
	if work.GetID() == "" {
		work.InitObject()
	}

	// persist durable work
	_, err = s.CRUD().CreateDup(sctx, work, true)
	if err != nil {
		c.SetLoggerField("work_reference_id", work.GetReferenceId())
		c.SetMessage("failed to save work in database")
		return c.SetError(err)
	}

	if postMode != SCHEDULE {

		if ctx.DbTransaction() != nil {
			c.Logger().Error("incompatible mode for calling inside transaction", nil)
			return nil
		}

		// nudge: wake the subscriber to call ClaimDueWorks; the DB lease decides ownership
		err = s.invoker(sctx, work, postMode, tenancy...)
		if err != nil {
			c.SetMessage("failed to invoke work")
			return err
		}
	} else {
		// For SCHEDULE mode, wake the scan goroutine so it picks up the row on the
		// next ClaimDueWorks pass rather than waiting the full PERIOD interval.
		// Safe inside a DB transaction: Wake only signals the rescan channel; it
		// does not touch the work object and ClaimDueWorks reads only committed rows.
		s.Wake()
	}

	// done
	return nil
}

func (s *WorkSchedule[T]) RemoveWork(sctx context.Context, referenceId string, refernecType string) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.AddWork")
	defer ctx.TraceOutMethod()

	// delete from database
	err = s.CRUD().DeleteByFields(sctx, db.Fields{"reference_id": referenceId, "reference_type": refernecType}, s.workBuilder())
	if err != nil {
		c.SetLoggerField("work_reference_id", referenceId)
		c.SetMessage("failed to delete work from database")
		return c.SetError(err)
	}

	// done
	return nil
}

func (s *WorkSchedule[T]) SetOverrideDb(db db.DB) {
	s.db = db
}

// ClaimDueWorks atomically leases the next batch of works that are due for
// processing (next_time <= now) and returns them. Each returned work has its
// next_time pushed HOLD_WORK_SECONDS into the future inside the claim
// transaction, so other schedulers polling the same database will not pick it up
// until the lease expires or the work is rescheduled/deleted after processing.
//
// A work that was claimed by another instance between the initial unlocked read
// and acquiring its row lock is re-checked for due-ness under the lock and
// skipped, so a given work is handed out to at most one instance per cycle.
// Returns an empty slice when nothing is due or the bucket is full.
func (s *WorkSchedule[T]) ClaimDueWorks(sctx context.Context) ([]T, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.ClaimDueWorks")
	defer ctx.TraceOutMethod()

	// prepare filter
	now := time.Now()
	filter := db.NewFilter()
	filter.SetSorting("next_time", db.SORT_ASC)
	filter.AddInterval("next_time", nil, now)

	// limit the batch to the remaining bucket capacity
	filter.Limit = s.BUCKET_SIZE - int(s.runningWorkCount.Load()) - int(s.workQueueSize.Load())
	if filter.Limit <= 0 {
		c.Logger().Info("all bucket size is used, skipping")
		return nil, nil
	}

	// read and lease due works in a single transaction
	var works []T
	handler := func() error {

		var works1 []T
		_, err := s.CRUD().List(sctx, filter, &works1)
		if err != nil {
			c.SetMessage("failed to read works from database 1")
			return err
		}

		// hold works
		nextTime := now.Add(time.Second * time.Duration(s.HOLD_WORK_SECONDS))
		workIds := []string{}
		for _, w := range works1 {
			dbWork := s.workBuilder()
			found, err := s.CRUD().ReadForUpdate(sctx, db.Fields{"id": w.GetID()}, dbWork)
			if err != nil {
				c.SetMessage("failed to read work for hold from database")
				return err
			}
			// Re-check due-ness under the row lock: another instance may have
			// claimed (held) this work between our unlocked List above and
			// acquiring the row lock here. Skip it if it is no longer due,
			// otherwise two instances would process the same work concurrently.
			if found && !dbWork.GetNextTime().After(now) {
				err = s.CRUD().Update(sctx, dbWork, db.Fields{"next_time": nextTime, "next_time_set": false})
				if err != nil {
					c.SetLoggerField("work_reference_id", dbWork.GetReferenceId())
					c.SetMessage("failed to hold work in database")
					return err
				}
				workIds = append(workIds, dbWork.GetID())
			}
		}

		// nothing left to claim (all candidates were taken by other instances)
		if len(workIds) == 0 {
			return nil
		}

		// read updated works
		f := db.NewFilter()
		f.AddFieldIn("id", utils.ListInterfaces(workIds...)...)
		_, err = s.CRUD().List(sctx, f, &works)
		if err != nil {
			c.SetMessage("failed to read works from database 2")
			return err
		}

		// done
		return nil
	}
	err := op_context.ExecDbTransaction(sctx, handler)
	if err != nil {
		return nil, c.SetError(err)
	}

	if len(works) > 0 {
		ids := make([]string, 0, len(works))
		for _, w := range works {
			ids = append(ids, w.GetReferenceType()+":"+w.GetReferenceId())
		}
		ctx.Logger().Debug("WorkSchedule: claimed due works", map[string]interface{}{
			"scheduler": s.name,
			"count":     len(works),
			"works":     ids,
		})
	}

	return works, nil
}

func (s *WorkSchedule[T]) readWorks(schedulerCtx context.Context) {

	readBatch := func() bool {

		ctx, sctx := default_op_context.BackgroundOpContext(s.App(), s.name)
		if s.db != nil {
			ctx.SetOverrideDb(s.db)
		}
		ctx.SetWriteCloseLog(s.LOG_EMPTY_WORKS)
		defer ctx.Close(sctx)

		works, err := s.ClaimDueWorks(sctx)
		if err != nil {
			return false
		}

		// stop cycle if there are no works
		if len(works) == 0 {
			return false
		}

		// enqueu works to workers
		for _, work := range works {
			if !s.enqueuWork(sctx, work, nil) {
				return false
			}
		}

		return true
	}

	// read works
	for {

		select {
		case <-schedulerCtx.Done():
			return

		case <-s.stop:
			return

		default:
			if !readBatch() {
				return
			}
		}
	}
}

func (s *WorkSchedule[T]) Run(ctx context.Context) {

	for i := 0; i < s.PARALLEL_JOBS; i++ {
		s.wg.Add(1)
		go s.worker(ctx)
	}

	// Poll period for re-scanning the database for works that became due while
	// the scheduler was idle. Without this the reader is only re-triggered when
	// a worker finishes an item, so future-scheduled works would never be picked
	// up once the queue drains.
	period := s.PERIOD
	if period <= 0 {
		period = 5
	}

	go func() {

		ticker := time.NewTicker(time.Second * time.Duration(period))

		defer func() {
			ticker.Stop()
			close(s.queue)
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case <-s.stop:
				return

			case <-ticker.C:
				s.readWorks(ctx)

			case _, ok := <-s.readNextWorks:
				if !ok {
					return
				}
				s.readWorks(ctx)
			}
		}
	}()

	s.readNextWorks <- struct{}{}
}

func (s *WorkSchedule[T]) DoWork(sctx context.Context, work T) error {

	// setup
	var err error
	releaseWork := false
	ctx := op_context.OpContext[op_context.Context](sctx)
	ctx.SetLoggerField("work_reference_id", work.GetReferenceId())
	c := ctx.TraceInMethod("WorkSchedule.DoWork")
	onExit := func() {

		if releaseWork {
			s.ReleaseWork(sctx, work)
		}
		if err != nil {
			c.SetError(err)
		}

		ctx.TraceOutMethod()
	}
	defer onExit()

	// check work runner
	if s.workRunner == nil {
		err = errors.New("invalid work runner")
		return err
	}

	// acquire work
	err = s.AcquireWork(sctx, work)
	if err != nil {
		if errors.Is(err, ErrWorkLocked) {
			// work is already being processed elsewhere, skip it silently
			err = nil
		}
		return err
	}
	releaseWork = true

	// run work
	work.ResetNextTime()
	done, err := s.workRunner.Run(sctx, work)
	s.SetNextWorkTime(work)
	updateProcessedWork := func() error {

		// read work from database
		dbWork := s.workBuilder()
		found, err := s.CRUD().ReadForUpdate(sctx, db.Fields{"id": work.GetID()}, dbWork)
		if err != nil {
			c.SetMessage("failed to read processed work from database")
			return err
		}
		if !found {
			// no need to update work in database
			return nil
		}

		// if done then just delete work
		if done {
			err = s.CRUD().Delete(sctx, dbWork)
			if err != nil {
				c.SetMessage("failed to delete processed work from database")
				return err
			}
			return nil
		}

		// set next time
		f := db.NewFilter()
		f.AddField("id", work.GetID())
		f.AddField("next_time_set", false)
		err = s.CRUD().UpdateWithFilter(sctx, s.workBuilder(), f, db.Fields{"next_time": work.GetNextTime(), "next_time_set": true})
		if err != nil {
			c.SetMessage("failed to save next work time in database")
			return err
		}

		// done
		return nil
	}
	err1 := op_context.ExecDbTransaction(sctx, updateProcessedWork)
	if err1 != nil {
		c.Logger().Error("failed to update processed work", err1)
		if err == nil {
			err = err1
		}
		return err
	}

	// done
	return nil
}

func (s *WorkSchedule[T]) InvokeWork(sctx context.Context, work T, postMode PostMode, tenancy ...multitenancy.Tenancy) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.InvokeWork")
	defer ctx.TraceOutMethod()

	switch postMode {
	case DIRECT:
		// TODO support multitenancy
		err := s.DoWork(sctx, work)
		if err != nil {
			return c.SetError(err)
		}
	case QUEUED:
		s.enqueuWork(sctx, work, tenancy...)
	}
	return nil
}

func (s *WorkSchedule[T]) UpdateWorkNextTime(sctx context.Context, work T, tenancy ...multitenancy.Tenancy) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("WorkSchedule.UpdateWorkNextTime")
	defer ctx.TraceOutMethod()

	err := s.CRUD().Update(sctx, work, db.Fields{"next_time": work.GetNextTime(), "next_time_set": true})
	if err != nil {
		c.SetMessage("failed to save next work time in database")
		return c.SetError(err)
	}

	return nil
}

func (s *WorkSchedule[T]) worker(schedulerCtx context.Context) {

	defer s.wg.Done()

	doWork := func(work workItem[T]) {
		s.workQueueSize.Add(-1)

		if work.tenancy != nil {
			ctx, sctx := app_with_multitenancy.BackgroundOpContext(s.App(), work.tenancy, s.name)
			s.DoWork(sctx, work.work)
			ctx.Close(sctx, "Served queue work")
		} else {
			ctx, sctx := default_op_context.BackgroundOpContext(s.App(), s.name)
			if s.db != nil {
				ctx.SetOverrideDb(s.db)
			}
			s.DoWork(sctx, work.work)
			ctx.Close(sctx, "Served queue work")
		}

		select {
		case s.readNextWorks <- struct{}{}:
		default:
		}
	}

	for {

		select {
		case <-schedulerCtx.Done():
			return

		case <-s.stop:
			return

		case work, ok := <-s.queue:
			if !ok {
				return
			}
			doWork(work)
		}
	}
}

func (s *WorkSchedule[T]) enqueuWork(ctx context.Context, work T, tenancy ...multitenancy.Tenancy) bool {
	s.workQueueSize.Add(1)

	select {
	case s.queue <- workItem[T]{work: work, tenancy: utils.OptionalArg(nil, tenancy...)}:
	case <-ctx.Done():
		return false
	case <-s.stop:
		return false
	}

	return true
}
