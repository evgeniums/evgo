package app_context

import (
	"fmt"
	"time"

	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/event_dispatcher"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type BuildConfig struct {
	Version    string
	Time       string
	Revision   string
	Label      string
	ConfigFile string // path to the config file used to initialise the application
}

// Print writes the build configuration to stdout, e.g. in response to a --version/-v switch.
func (b *BuildConfig) Print() {
	fmt.Printf("Label: %s\nVersion: %s\nRevision: %s\nBuild time: %s\n", b.Label, b.Version, b.Revision, b.Time)
}

type Context interface {
	logger.WithLogger
	config.WithCfg
	db.WithDB

	BuildConfig() *BuildConfig

	Cache() cache.Cache
	Validator() validator.Validator

	Testing() bool
	TestParameters() map[string]interface{}
	SetTestParameter(key string, value interface{})
	GetTestParameter(key string) (interface{}, bool)

	AppInstance() string
	Application() string
	Hostname() string

	EventDispatcher() event_dispatcher.Dispatcher

	Close()
}

var Timezone = "UTC"
var TimeLocationOs *time.Location

func SetTimeZone(timezone ...string) error {

	tz := utils.OptionalArg(Timezone, timezone...)

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	time.Local = loc
	Timezone = tz

	return nil
}
