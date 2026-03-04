package app_context

import (
	"time"

	"github.com/evgeniums/go-utils/pkg/cache"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/db"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
)

type BuildConfig struct {
	Version  string
	Time     string
	Revision string
	Label    string
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
