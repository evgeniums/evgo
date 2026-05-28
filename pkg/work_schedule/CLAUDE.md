# work_schedule

Database-backed deferred/recurring job queue with pool-aware dispatch. Works are persisted as GORM rows, claimed via an atomic DB lease, and protected by a cache lock so each item is processed by at most one worker at a time.

## Minimal integration recipe

### 1. Define a Work type

Embed `WorkBase` in a GORM model and register the table:

```go
type MyWork struct {
    work_schedule.WorkBase
}
func NewMyWork() *MyWork { return &MyWork{} }
```

Register `&MyWork{}` in your `dbModels` list so the table is migrated.
For multi-tenant deployments embed `WorkInTenancyBase` instead.

### 2. Implement WorkRunner

```go
type MyRunner struct{ /* your dependencies */ }

func (r *MyRunner) Run(sctx context.Context, w *MyWork) (done bool, _ error) {
    switch w.GetReferenceType() {
    case "type-a":
        // ... do work identified by w.GetReferenceId() ...
        return true, nil   // true = delete the row when done
    }
    return false, nil      // false = reschedule for next interval
}
```

The runner must be **idempotent**: the scheduler delivers at-least-once (a crash during processing causes the held row to become due again after `HOLD_WORK_SECONDS`).

### 3. Create and start the scheduler

```go
scheduler := work_schedule.NewWorkSchedule[*MyWork]("my_work",
    work_schedule.Config[*MyWork]{
        WorkBuilder: NewMyWork,
        WorkRunner:  myRunner,          // can also be set later with SetRunner
        WorkInvoker: publisher.InvokeWork, // omit for single-process deployments
        Locker:      myLocker,          // omit to default to in-memory locker
    })
err = scheduler.Init(app, "my_work_scheduler") // reads config from this path
// ...
scheduler.Run(ctx)        // starts worker pool + periodic DB scan
defer scheduler.Shutdown(context.Background())
```

### 4. Post work from business logic

```go
// delayed first run (SCHEDULE mode, persisted)
w := scheduler.NewWork(entityId, "type-a")
w.SetInitialDelay(600) // seconds; omit to use INVOKATION_INTERVAL_SECONDS
err = scheduler.PostWork(sctx, w, work_schedule.SCHEDULE)

// immediate run via nudge (QUEUED mode, persisted)
err = scheduler.PostWork(sctx, scheduler.NewWork(id, "type-b"), work_schedule.QUEUED)

// cancel a pending work
err = scheduler.RemoveWork(sctx, entityId, "type-a")
```

## Post modes

| Mode | Behaviour |
|---|---|
| `SCHEDULE` | Persists with a future `next_time` (using `InitialDelay` or `INVOKATION_INTERVAL_SECONDS`). Poll loop picks it up. Normal path for recurring/delayed work. |
| `QUEUED` | Persists with `next_time=now`, then sends a nudge to wake the subscriber. Work is claimed via DB lease by the first available worker. |
| `DIRECT` | Same as `QUEUED` but the nudge recipient runs the work inline rather than enqueuing it. Avoid inside a DB transaction. |

`QUEUED` and `DIRECT` must not be called inside an open `DbTransaction` — `PostWork` logs an error and no-ops if `ctx.DbTransaction() != nil`.

## Pool deployment (multi-instance)

In a pool each instance needs a **publisher** (sends nudges on post) and a **subscriber** (receives nudges, calls `ClaimDueWorks`):

```go
// On every instance:
publisher := work_schedule.NewPoolWorkPublisher[*MyWork](app.Pubsub(), "my_work_topic")
scheduler := work_schedule.NewWorkSchedule[*MyWork]("my_work",
    work_schedule.Config[*MyWork]{
        WorkBuilder: NewMyWork,
        WorkInvoker: publisher.InvokeWork,
        Locker:      redis_cache.NewLocker(redisCache), // REQUIRED for pool safety
    })
scheduler.Init(app, "my_work_scheduler")

// On processor instances only:
subscriber := work_schedule.NewPoolSubscriber[*MyWork](tenancies, scheduler, "my_processor")
subscriber.Init(sctx, app.Pubsub(), "my_work_topic")
scheduler.SetRunner(myRunner)
scheduler.Run(ctx)
```

**The `Locker` must be a distributed lock (Redis) in pool deployments.** The default in-memory locker only protects within a single process. Without a shared locker, two instances can both claim the same work through the DB lease race window and both execute it. Use `redis_cache.NewLocker(redisCache)` — the Redis client is available via `app_default` when `redis_cache` is configured.

## no-db works

Set `work.SetNoDb(true)` to skip DB persistence entirely. The work payload is sent directly over pubsub to the subscriber, which invokes the runner without any DB lease or lock.

Use only when: the work is transient (no retry semantics needed), the entity data is already in the DB (only `ReferenceId`+`ReferenceType` are needed to route), and the runner is idempotent. The sole guard against duplicate execution is the cache lock — which is in-memory and process-local unless a distributed locker is configured.

Typical use: webhook gateway → background processor handoff.

## Configuration

Read from the JSON config path passed to `Init` (keys are lowercase field names):

| Key | Default | Meaning |
|---|---|---|
| `parallel_jobs` | 8 | Worker goroutines per scheduler instance |
| `bucket_size` | 32 | Max works claimed + in-flight at once |
| `invokation_interval_seconds` | 300 | Reschedule interval when `InitialDelay` is 0 |
| `hold_work_seconds` | 900 | How long a claimed row is leased (pushed into the future) |
| `lock_ttl_seconds` | 900 | Cache lock TTL; keep ≥ `hold_work_seconds` |
| `period` | 5 | Ticker interval (seconds) for rescanning the DB |
| `log_empty_works` | false | Log when the poll cycle finds nothing |

## Key invariants

- **`done=true`** from the runner deletes the row; **`done=false`** reschedules it at `now + INVOKATION_INTERVAL_SECONDS` (or `InitialDelay` on first run only — `InitialDelay` is not persisted and is not used on re-runs).
- **`LOCK_TTL_SECONDS` must be ≥ `HOLD_WORK_SECONDS`** so the cache lock outlives the DB lease. The defaults are aligned; do not set them independently without understanding the interaction.
- **`SCHEDULE` inside a transaction is safe**; `QUEUED`/`DIRECT` are not — they trigger an immediate nudge/invoke which would run concurrently with the uncommitted transaction.
- The DB hold (lease) is the primary correctness guard; the cache lock is the secondary guard. Both must work for pool safety. A missing distributed locker degrades correctness but does not break single-process deployments.

## Test reference

`evgo/test/work_schedule_test/work_schedule_test.go` — full integration tests covering claim/lease semantics, concurrent claim, locking, all post modes, ticker, and the nudge subscriber path.
