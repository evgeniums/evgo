package app_default

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/cache"
	"github.com/evgeniums/evgo/pkg/cache/inmem_cache"
	"github.com/evgeniums/evgo/pkg/cache/redis_cache"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/config_viper"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/db/db_gorm"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/logger/logger_logrus"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
	"github.com/evgeniums/evgo/pkg/validator/validator_playground"
)

var Version = "development"
var Time = "unknown"
var Revision = "unknown"
var Label = "white"

type contextConfig struct {
	TESTING      bool
	APP_INSTANCE string
	HOSTNAME     string

	APPLICATION string
}

type WithInitGormDb interface {
	InitDB(configPath string, gormDbConnector ...*db_gorm.DbConnector) error
}

type Context struct {
	logger.WithLoggerBase
	config.WithCfgBase

	buildConfig *app_context.BuildConfig

	db           *db_gorm.GormDB
	validator    *validator_playground.PlaygroundValdator
	cache        cache.Cache
	inmemCache   *inmem_cache.InmemCache[string]
	redisCache   *redis_cache.RedisCache
	logrusLogger *logger_logrus.LogrusLogger

	contextConfig

	testValues  map[string]interface{}
	initialized bool
}

func (c *Context) BuildConfig() *app_context.BuildConfig {
	return c.buildConfig
}

func (c *Context) Config() interface{} {
	return &c.contextConfig
}

func (c *Context) Db() db.DB {
	return c.db
}

func (c *Context) Cache() cache.Cache {
	return c.cache
}

func (c *Context) Validator() validator.Validator {
	return c.validator
}

func (c *Context) Testing() bool {
	return c.TESTING
}

func (c *Context) AppInstance() string {
	return c.APP_INSTANCE
}

func Application() string {
	proc, _ := os.Executable()
	return filepath.Base(proc)
}

func (c *Context) Application() string {
	if c.APPLICATION != "" {
		return c.APPLICATION
	}
	return Application()
}

func (c *Context) TestParameters() map[string]interface{} {
	return c.testValues
}

func (c *Context) SetTestParameter(key string, value interface{}) {
	c.testValues[key] = value
}

func (c *Context) GetTestParameter(key string) (interface{}, bool) {
	value, ok := c.testValues[key]
	return value, ok
}

type AppConfig struct {
	Cache cache.Cache
}

func (a *AppConfig) GetCache() cache.Cache {
	return a.Cache
}

type AppConfigI interface {
	GetCache() cache.Cache
}

func New(buildConfig *app_context.BuildConfig, appConfig ...AppConfigI) *Context {

	if app_context.TimeLocationOs == nil {
		app_context.TimeLocationOs = time.Local
	}

	c := &Context{}

	if buildConfig != nil {
		c.buildConfig = buildConfig
	} else {
		c.buildConfig = &app_context.BuildConfig{
			Version:  Version,
			Time:     Time,
			Revision: Revision,
			Label:    Label,
		}
	}

	c.validator = validator_playground.New()

	if len(appConfig) != 0 {
		c.cache = appConfig[0].GetCache()
	}

	c.logrusLogger = logger_logrus.New()
	c.WithLoggerBase.Init(c.logrusLogger)

	return c
}

func (c *Context) InitWithArgs(configFile string, args []string, configType ...string) error {

	if c.initialized {
		return nil
	}

	app_context.SetTimeZone()

	// load configuration
	fmt.Printf("Application %s using configuration file %s\n", c.Application(), configFile)
	err := c.initConfig(configFile)
	if err != nil {
		return c.Logger().PushFatalStack("failed to load application configuration", err)
	}

	// load command line arguments
	err = config.LoadArgs(c.Cfg(), args)
	if err != nil {
		return c.Logger().PushFatalStack("failed to override configuration parameters", err)
	}

	// setup logger
	logConfigPath := "logger"
	err = c.initLog(logConfigPath)
	if err != nil {
		return c.Logger().PushFatalStack("failed to init application logger", err)
	}
	log := c.Logger()

	// log build version
	log.Info("Build configuration", logger.Fields{"build_time": c.buildConfig.Time, "package_version": c.buildConfig.Version, "git_revision": c.buildConfig.Revision})
	fmt.Printf("Build configuration: build_time=%s, package_version=%s, git_revision=%s\n", c.buildConfig.Time, c.buildConfig.Version, c.buildConfig.Revision)

	// log logger configuration
	object_config.Info(log, c.logrusLogger, logConfigPath)

	// load top level application configuration
	err = object_config.LoadLogValidate(c.Cfg(), log, c.validator, c, "")
	if err != nil {
		return log.PushFatalStack("failed to init application configuration", err)
	}

	// setup testing
	if c.Testing() {
		log.Info("Running in test mode")
		c.testValues = make(map[string]interface{})
	}

	// init cache
	if c.cache == nil {
		redisCacheConfigPath := redis_cache.RedisCacheConfigPath
		if c.Cfg().IsSet(redisCacheConfigPath) {
			log.Info("using Redis cache as application cache")
			c.redisCache = redis_cache.NewCache()
			err = c.redisCache.Init(c.Cfg(), c.Logger(), c.Validator(), redisCacheConfigPath)
			if err != nil {
				return log.PushFatalStack("failed to init redis cache", err)
			}
			c.cache = cache.New(c.redisCache)
		} else {
			log.Info("using in-memory cache as application cache")
			c.inmemCache = inmem_cache.New[string]()
			c.cache = cache.New(c.inmemCache)
			c.inmemCache.Start()
		}
	}

	// done
	c.initialized = true
	return nil
}

func (c *Context) Init(configFile string, configType ...string) error {
	return c.InitWithArgs(configFile, nil, configType...)
}

func (c *Context) Close() {
	if c.db != nil {
		c.db.Close()
	}
	if c.inmemCache != nil {
		c.inmemCache.Stop()
	}
	if c.redisCache != nil {
		c.redisCache.Stop()
	}
}

func (c *Context) initConfig(configFile string, configType ...string) error {

	if configFile == "" {
		return errors.New("configuration file not specified")
	}

	v := config_viper.New()
	c.SetCfg(v)
	err := v.LoadFile(configFile, configType...)
	if err != nil {
		return err
	}
	object_config.Load(v, c, "")
	return nil
}

func (c *Context) initLog(configPath string) error {
	return c.logrusLogger.Init(c.Cfg(), c.validator, configPath)
}

func (c *Context) InitDB(configPath string, gormDbConnector ...*db_gorm.DbConnector) error {
	if c.db != nil {
		return nil
	}
	d := db_gorm.New(gormDbConnector...)
	c.db = d
	return d.Init(c, c.Cfg(), c.validator, configPath)
}

func (c *Context) Hostname() string {
	if c.HOSTNAME != "" {
		return c.HOSTNAME
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknow"
	}
	return hostname
}

func ConfigFile(defaultConfigFile ...string) string {
	appPath := Application()
	configPath := fmt.Sprintf("%s.jsonc", appPath[:len(appPath)-len(filepath.Ext(appPath))])
	configPath = utils.OptionalString(configPath, defaultConfigFile...)

	configFile := flag.String("config", configPath, "Configuration file")
	flag.Parse()
	fmt.Printf("Using config file %v\n", *configFile)

	return *configFile
}
