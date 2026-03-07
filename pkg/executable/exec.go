package executable

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/multitenancy_background_app"
	"github.com/evgeniums/evgo/pkg/op_context"
)

var ForceDefaultConfigFile bool
var InitBaseApp bool
var InitBaseAppDb bool

type Instance interface {
	multitenancy_background_app.MainRunner
	Init(app app_with_multitenancy.AppWithMultitenancy, opCtx op_context.Context) error
}

func New(instance Instance, defaultConfigFile string, buildConfig *app_context.BuildConfig, tenancyDbModels *multitenancy.TenancyDbModels, appConfig ...app_with_multitenancy.AppConfigI) *multitenancy_background_app.Main {

	initInstance := func(app app_with_multitenancy.AppWithMultitenancy, opCtx op_context.Context) (multitenancy_background_app.MainRunner, error) {
		err := instance.Init(app, opCtx)
		if err != nil {
			return nil, err
		}

		return instance, nil
	}

	return multitenancy_background_app.New(buildConfig, tenancyDbModels, &multitenancy_background_app.RunnerConfig{RunnerBuilder: initInstance, DefaultConfigFile: defaultConfigFile, ForceDefaultConfigFlag: ForceDefaultConfigFile, InitBaseApp: InitBaseApp, InitBaseAppDb: InitBaseAppDb}, appConfig...)
}

func Exec(instance Instance, buildConfig *app_context.BuildConfig, tenancyDbModels *multitenancy.TenancyDbModels, appConfig ...app_with_multitenancy.AppConfigI) {
	main := New(instance, "", buildConfig, tenancyDbModels, appConfig...)
	main.Exec()
}
