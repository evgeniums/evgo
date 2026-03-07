package background_worker

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/evgeniums/evgo/pkg/logger"
	finish "github.com/evgeniums/go-finish-service"
	"github.com/markphelps/optional"
)

type Runner interface {
	Shutdown(ctx context.Context) error
}

type RunnerConfig struct {
	Name    optional.String
	Timeout optional.Int
}

const DefaultTimeout = 10 * time.Second

var DefaultSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}

type FinisherMainConfig struct {

	// Timeout is the maximum amount of time to wait for
	// still running runners requests to finish,
	// when the shutdown signal was received for each runner.
	//
	// It defaults to DefaultTimeout which is 10 seconds.
	//
	// The timeout can be overridden on a per-server basis with passing the
	// WithTimeout() option to Add() while adding the runner.
	Timeout time.Duration

	// Signals can be used to change which signals finish catches to initiate
	// the shutdown.
	// It defaults to DefaultSignals which contains SIGINT and SIGTERM.
	Signals []os.Signal
}

type FinisherConfig struct {
	FinisherMainConfig

	// Log can be set to change where finish logs to.
	Logger logger.Logger
}

type Finisher interface {
	AddRunner(runner Runner, config ...*RunnerConfig)
	Wait()
	Shutdown(ctx context.Context) error
	RemoveRunner(name string)
}

type finisherLogger struct {
	logger logger.Logger
}

func (l *finisherLogger) Infof(format string, v ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, v...))
}

func (l *finisherLogger) Errorf(format string, v ...interface{}) {
	l.logger.ErrorRaw(fmt.Sprintf(format, v...))
}

type FinisherBase struct {
	finisher *finish.Finisher
}

func NewFinisher(config ...*FinisherConfig) *FinisherBase {

	f := &FinisherBase{}
	f.finisher = finish.New()

	if len(config) != 0 {
		cfg := config[0]
		f.finisher.Timeout = cfg.Timeout
		f.finisher.Signals = cfg.Signals
		if cfg.Logger != nil {
			l := &finisherLogger{logger: cfg.Logger}
			f.finisher.Log = l
		}
	} else {
		f.finisher.Signals = DefaultSignals
		f.finisher.Timeout = DefaultTimeout
		f.finisher.Log = finish.DefaultLogger
	}

	return f
}

func (f *FinisherBase) AddRunner(runner Runner, config ...*RunnerConfig) {

	if len(config) == 0 {
		f.finisher.Add(runner)
		return
	}

	cfg := config[0]
	options := make([]finish.Option, 0)
	if cfg.Name.Present() {
		options = append(options, finish.WithName(cfg.Name.MustGet()))
	}
	if cfg.Timeout.Present() {
		options = append(options, finish.WithTimeout(time.Second*time.Duration(cfg.Timeout.MustGet())))
	}

	f.finisher.Add(runner, options...)
}

func (f *FinisherBase) Shutdown(ctx context.Context) error {
	f.finisher.Trigger()
	return nil
}

func (f *FinisherBase) Wait() {
	f.finisher.Wait()
}

func (f *FinisherBase) RemoveRunner(name string) {
	f.finisher.Remove(name)
}
