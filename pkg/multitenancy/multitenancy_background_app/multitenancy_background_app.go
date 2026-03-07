package multitenancy_background_app

import (
	"flag"
	"fmt"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/background_worker"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/db/db_gorm"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/markphelps/optional"
)

type MainRunner interface {
	Run(fin background_worker.Finisher)
}

type BuildMainRunner = func(app app_with_multitenancy.AppWithMultitenancy, opCtx op_context.Context) (MainRunner, error)

type RunnerConfig struct {
	*background_worker.FinisherMainConfig
	RunnerBuilder          BuildMainRunner
	DefaultConfigFile      string
	ForceDefaultConfigFlag bool
	InitBaseApp            bool
	InitBaseAppDb          bool
}

type Main struct {
	runner   MainRunner
	Finisher background_worker.Finisher
	App      app_with_multitenancy.AppWithMultitenancy
}

func ConfigFile(defaultConfigFile ...string) string {
	return app_default.ConfigFile(defaultConfigFile...)
}

func New(buildConfig *app_context.BuildConfig, tenancyDbModels *multitenancy.TenancyDbModels, runnerConfig *RunnerConfig, appConfig ...app_with_multitenancy.AppConfigI) *Main {

	// get name of configuration file
	configFile := runnerConfig.DefaultConfigFile
	if !runnerConfig.ForceDefaultConfigFlag {
		configFile = ConfigFile(runnerConfig.DefaultConfigFile)
	}

	// init db gorm models
	db_gorm.NewModelStore(true)

	// init app context
	app := app_with_multitenancy.NewApp(buildConfig, tenancyDbModels, appConfig...)
	var initOpCtx op_context.Context
	var err error
	if runnerConfig.InitBaseApp {
		err = app.Context.InitWithArgs(configFile, flag.Args())
		if err != nil && runnerConfig.InitBaseAppDb {
			err = app.Context.InitDB("db")
		}
		initOpCtx = default_op_context.NewAppInitContext(app)
	} else {
		initOpCtx, err = app.InitWithArgs(configFile, flag.Args())
	}

	if err != nil {
		if initOpCtx != nil {
			initOpCtx.Close()
		}
		app_context.AbortFatal(app, "failed to init application context", err)
	}

	// create main runner
	runner, err := runnerConfig.RunnerBuilder(app, initOpCtx)
	initOpCtx.Close()
	if err != nil {
		app_context.AbortFatal(app, "failed to init main runner", err)
	}

	// create finisher
	finisherConfig := &background_worker.FinisherConfig{Logger: app.Logger()}
	if runnerConfig.FinisherMainConfig != nil {
		finisherConfig.FinisherMainConfig = *runnerConfig.FinisherMainConfig
	}
	fin := background_worker.NewFinisher(finisherConfig)
	fin.AddRunner(app.Pubsub(), &background_worker.RunnerConfig{Name: optional.NewString("pubsub")})

	// return main
	return &Main{runner: runner, App: app, Finisher: fin}
}

func (m *Main) Exec() {

	// log
	startedMsg := fmt.Sprintf("%s started", m.App.Application())
	m.App.Logger().Info(startedMsg)
	fmt.Println(startedMsg)

	// run runner
	m.runner.Run(m.Finisher)

	// wait for signals
	m.Finisher.Wait()

	// done
	finishedMsg := fmt.Sprintf("%s finished", m.App.Application())
	m.App.Logger().Info(finishedMsg)
	m.App.Close()
	db.Databases().CloseAll()
	fmt.Println(finishedMsg)
}
