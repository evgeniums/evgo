package object_config

import (
	"fmt"
	"reflect"

	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
)

func Info(log logger.Logger, obj Object, configPath ...string) {
	Log(log, utils.OptionalArg("", configPath...), obj)
}

func Debug(log logger.Logger, obj Object, configPath ...string) {
	Log(log, utils.OptionalArg("", configPath...), obj, logger.DebugLevel)
}

func logObject(log logger.Logger, configPath string, obj reflect.Value, logLevel ...logger.Level) {

	level := utils.OptionalArg(logger.InfoLevel, logLevel...)

	typ := obj.Type()
	if typ.Kind() == reflect.Ptr {
		typ = obj.Elem().Type()
	}

	for i := 0; i < typ.NumField(); i++ {

		field := typ.Field(i)

		var value reflect.Value
		if obj.Type().Kind() == reflect.Ptr {
			value = obj.Elem().Field(i)
		} else {
			value = obj.Field(i)
		}

		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			logObject(log, configPath, value.Addr(), logLevel...)
		} else if field.Type.Kind() != reflect.Map {
			keyPath := Key(configPath, field.Name)
			formatValue := func() string {

				val := ""
				_, ok := field.Tag.Lookup("mask")
				if ok {
					val = "********"
				} else {
					val = fmt.Sprintf("%v", value)
				}

				return val
			}
			LogParameter(log, keyPath, formatValue(), level)
		}
	}
}

func Log(log logger.Logger, configPath string, obj Object, logLevel ...logger.Level) {
	logObject(log, configPath, reflect.ValueOf(obj.Config()), logLevel...)
}

func LogParameter(log logger.Logger, key string, value string, logLevel logger.Level) {
	log.Log(logLevel, "Configuration", logger.Fields{"key": key, "value": value})
}
