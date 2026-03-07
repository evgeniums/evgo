package console_tool

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"strings"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/logger/logger_logrus"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/jessevdk/go-flags"
)

type MainOptions struct {
	ConfigFile   string `short:"c" long:"config" description:"Configuration file"`
	ConfigFormat string `short:"f" long:"config-format" description:"Format of configuration file" default:"json"`
	NoInitDb     bool   `short:"w" long:"without-database" description:"Don't initialize database"`
	Setup        bool   `short:"s" long:"setup" description:"Use minimal application features for initial setup"`
	DbSection    string `short:"d" long:"database-section" description:"Database section in configuration file" default:"db"`
	Tenancy      string `short:"t" long:"tenancy" description:"Tenancy to invoke the command in. Can be either tenancy's ID or in the form of customer_name/role."`
	Args         string `short:"a" long:"args" description:"Additional configuration arguments."`
}

type Dummy struct{}
type ContextBulder = func(group string, command string) multitenancy.TenancyContext

type ConsoleUtility struct {
	Parser *flags.Parser
	App    app_context.Context
	Opts   MainOptions
	Args   []string

	AppBuilder  AppBuilder
	BuildConfig *app_context.BuildConfig
}

type InitApp = func(app app_context.Context, configFile string, args []string, configType ...string) error

type AppBuilder interface {
	NewApp(buildConfig *app_context.BuildConfig) app_context.Context
	InitApp(app app_context.Context, configFile string, args []string, configType ...string) error
	NewSetupApp(buildConfig *app_context.BuildConfig) app_context.Context
	InitSetupApp(app app_context.Context, configFile string, args []string, configType ...string) error
	HasSetupApp() bool
	Tenancy(ctx op_context.Context, id string) (multitenancy.Tenancy, error)
	PoolController() pool.PoolController
}

func New(buildConfig *app_context.BuildConfig, appBuilder ...AppBuilder) *ConsoleUtility {
	c := &ConsoleUtility{}
	c.Parser = flags.NewParser(&c.Opts, flags.Default)

	c.BuildConfig = buildConfig
	if len(appBuilder) != 0 {
		c.AppBuilder = appBuilder[0]
	}
	return c
}

func (c *ConsoleUtility) Close() {
	if c.App != nil {
		c.App.Close()
	}
	db.Databases().CloseAll()
}

func (c *ConsoleUtility) InitCommandContext(group string, command string) multitenancy.TenancyContext {

	if c.Opts.Args != "" {
		c.Args = []string{}
		args := strings.Split(c.Opts.Args, " ")
		for _, arg := range args {
			pair := strings.Split(arg, "=")
			if len(pair) != 2 {
				app_context.AbortFatal(c.App, "invalid additional args", nil)
			}
			c.Args = append(c.Args, fmt.Sprintf("--%s", pair[0]))
			c.Args = append(c.Args, pair[1])
		}
	}

	if c.AppBuilder == nil || c.Opts.Setup && !c.AppBuilder.HasSetupApp() {

		fmt.Println("Using minimal application context for setup")

		app := app_default.New(c.BuildConfig)
		c.App = app

		err := app.InitWithArgs(c.Opts.ConfigFile, c.Args, c.Opts.ConfigFormat)
		if err != nil {
			app_context.AbortFatal(c.App, "failed to init context of minimal application", err)
		}
	} else if c.AppBuilder != nil && c.Opts.Setup && c.AppBuilder.HasSetupApp() {

		fmt.Println("Using setup application context")

		c.App = c.AppBuilder.NewSetupApp(c.BuildConfig)
		err := c.AppBuilder.InitSetupApp(c.App, c.Opts.ConfigFile, c.Args, c.Opts.ConfigFormat)
		if err != nil {
			app_context.AbortFatal(c.App, "failed to init context of setup application", err)
		}
	} else {

		fmt.Println("Using normal application context")

		c.App = c.AppBuilder.NewApp(c.BuildConfig)
		err := c.AppBuilder.InitApp(c.App, c.Opts.ConfigFile, c.Args, c.Opts.ConfigFormat)
		if err != nil {
			app_context.AbortFatal(c.App, "failed to init application context", err)
		}
	}

	if c.App.Cfg().GetString("logger.destination") != "stdout" {
		c.App.Cfg().Set("logger.destination", "stdout")
		consoleLogger := logger_logrus.New()
		err := consoleLogger.Init(c.App.Cfg(), c.App.Validator(), "logger")
		if err != nil {
			app_context.AbortFatal(c.App, "failed to init console logger", err)
		}
		teeLogger := logger.NewTee(c.App.Logger(), consoleLogger)
		c.App.SetLogger(teeLogger)
	}

	if !c.Opts.NoInitDb {
		initDbApp, ok := c.App.(app_default.WithInitGormDb)
		if !ok {
			app_context.AbortFatal(c.App,
				"failed to init database",
				c.App.Logger().PushFatalStack("invalid type of application", errors.New("failed to cast application to app with initdb")))
		}
		err := initDbApp.InitDB(c.Opts.DbSection)
		if err != nil {
			app_context.AbortFatal(c.App, "failed to init database", err)
		}
	}

	opCtx := multitenancy.NewInitContext(c.App, c.App.Logger(), c.App.Db())
	opCtx.SetName(fmt.Sprintf("%s.%s", group, command))
	errManager := &generic_error.ErrorManagerBase{}
	errManager.Init(http.StatusBadRequest)
	opCtx.SetErrorManager(errManager)

	userName := "unknown"
	u, e := user.Current()
	if e != nil || u == nil {
		fmt.Printf("failed to get OS user information: %s\n", e)
	} else {
		userName = u.Username
	}

	origin := default_op_context.NewOrigin(c.App)
	origin.SetUser(userName)
	origin.SetUserType("console")
	opCtx.SetOrigin(origin)

	if c.Opts.Tenancy != "" {
		tenancy, err := c.AppBuilder.Tenancy(opCtx, c.Opts.Tenancy)
		if err != nil {
			app_context.AbortFatal(c.App, "failed to find tenancy", err)
		}
		opCtx.SetTenancy(tenancy)
	}

	c.App.Logger().Info("Console command context", logger.Fields{"user": userName,
		"app":          origin.App(),
		"app_instance": origin.Name(),
		"source":       origin.Source(),
		"name":         opCtx.Name(),
		"context":      opCtx.ID(),
	})

	return opCtx
}

func (c *ConsoleUtility) Parse() {
	var err error
	c.Args, err = c.Parser.Parse()
	if err == nil && len(c.Args) != 0 {
		fmt.Printf("Additional args: %v\n", c.Args)
	}
	if err != nil {
		if err, ok := err.(*flags.Error); ok {
			if err.Type == flags.ErrHelp {
				os.Exit(0)
			}
			c.Parser.WriteHelp(os.Stdout)
		}
		app_context.ErrorLn("Failed")
		os.Exit(1)
	}
}

func (c *ConsoleUtility) AddCommandGroup(handler func(ctxBuilder ContextBulder, parser *flags.Parser) *flags.Command) *flags.Command {
	return handler(c.InitCommandContext, c.Parser)
}

func (c *ConsoleUtility) AddCommandSubgroup(parent *flags.Command, handler func(ctxBuilder ContextBulder, parent *flags.Command) *flags.Command) *flags.Command {
	return handler(c.InitCommandContext, parent)
}
