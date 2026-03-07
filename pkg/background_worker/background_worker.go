package background_worker

import (
	system_context "context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/go-condchan"
	"github.com/markphelps/optional"
)

const ContextUser string = "background_user"

type BackgroundStopper interface {
	IsStopped() bool
}

type BackgroundStopperStub struct {
	stopped atomic.Bool
}

func (b *BackgroundStopperStub) IsStopped() bool {
	return b.stopped.Load()
}

func (b *BackgroundStopperStub) Stop() {
	b.stopped.Store(true)
}

type JobRunner interface {
	RunJob()
	StopJob()
	SetStopper(stopper BackgroundStopper)
	Stopper() BackgroundStopper
}

type JobRunnerBase struct {
	stopper BackgroundStopper
}

func (j *JobRunnerBase) StopJob() {
}

func (j *JobRunnerBase) SetStopper(stopper BackgroundStopper) {
	j.stopper = stopper
}

func (j *JobRunnerBase) Stopper() BackgroundStopper {
	return j.stopper
}

type BackgroundWorker struct {
	logger.WithLoggerBase

	Period int
	Name   string

	CondChan *condchan.CondChan
	Finished chan bool
	Stopped  atomic.Bool
	Running  atomic.Bool

	JobRunner JobRunner
}

type WithBackgroundWorker interface {
	Worker() *BackgroundWorker
}

type WithBackgroundWorkerBase struct {
	WorkerInterface *BackgroundWorker
}

func (w *WithBackgroundWorkerBase) Worker() *BackgroundWorker {
	return w.WorkerInterface
}

func New(log logger.Logger, jobRunner JobRunner, period int, name ...string) *BackgroundWorker {
	b := &BackgroundWorker{JobRunner: jobRunner, Period: period}
	jobRunner.SetStopper(b)
	b.WithLoggerBase.Init(log)
	b.CondChan = condchan.New(&sync.Mutex{})
	b.Finished = make(chan bool, 1)
	b.Name = utils.OptionalArg("", name...)
	return b
}

func (w *BackgroundWorker) RunInBackground() {

	w.Running.Store(true)
	w.Stopped.Store(false)

	// run in go routine
	go func() {
		w.JobRunner.RunJob()
		if w.IsStopped() {
			w.Logger().Debug("Background worker: stopped after first run")
			w.Finished <- true
		}
		for {

			timeoutChan := time.After(time.Second * time.Duration(w.Period+1))
			br := new(bool)
			*br = false

			w.CondChan.L.Lock()
			w.CondChan.Select(func(c <-chan struct{}) {
				select {
				case <-c:
					w.Logger().Debug("Background worker: signal received")
					*br = true
				case <-timeoutChan:
					if !w.IsStopped() {
						// w.Log.LogDebug("Background worker: run job")
						w.JobRunner.RunJob()
					}
				}
			})
			w.CondChan.L.Unlock()

			if w.IsStopped() || *br {
				w.Logger().Debug("Background worker: break cycle")
				break
			}
		}
		w.Finished <- true
	}()
}

func (w *BackgroundWorker) Stop() {
	w.Logger().Info("Background worker: stopping...")
	if !w.Running.Load() {
		w.Logger().Info("Background worker: not running, quit")
		return
	}
	w.Stopped.Store(true)
	w.JobRunner.StopJob()
	w.CondChan.Broadcast()
	<-w.Finished
	w.Running.Store(false)
	w.Logger().Info("Background worker: finished")
}

func (w *BackgroundWorker) Shutdown(ctx system_context.Context) error {
	w.Stop()
	return nil
}

func (w *BackgroundWorker) Run(fin Finisher) {
	fin.AddRunner(w, &RunnerConfig{Name: optional.NewString(w.Name)})
	w.RunInBackground()
}

func (w *BackgroundWorker) IsStopped() bool {
	return w.Stopped.Load()
}
