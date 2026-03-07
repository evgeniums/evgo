package object_config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type Object interface {
	Config() interface{}
}

func KeyInt(path string, key int) string {
	k := fmt.Sprintf("%d", key)
	if path == "" {
		return k
	}
	return fmt.Sprintf("%s.%s", path, k)
}

func Key(path string, key string) string {
	if path == "" {
		return key
	}
	return fmt.Sprintf("%s.%s", path, key)
}

func Load(cfg config.Config, obj Object, configPath string) error {
	_, err := loadConfiguration(cfg, configPath, obj)
	return err
}

func LoadValidate(cfg config.Config, vld validator.Validator, obj Object, defaultPath string, optionalPath ...string) error {

	path := utils.OptionalArg(defaultPath, optionalPath...)

	loadConfiguration(cfg, path, obj)
	err := vld.Validate(obj.Config())
	if err != nil {
		return err
	}

	return nil
}

func LoadLogValidateApp(app app_context.Context, obj Object, defaultPath string, optionalPath ...string) error {
	return LoadLogValidate(app.Cfg(), app.Logger(), app.Validator(), obj, defaultPath, optionalPath...)
}

// Load, print to log and validate oject. Known bug: default values do not work poperly within arrays.
func LoadLogValidate(cfg config.Config, log logger.Logger, vld validator.Validator, obj Object, defaultPath string, optionalPath ...string) error {

	path := utils.OptionalArg(defaultPath, optionalPath...)

	_, err := loadConfiguration(cfg, path, obj)
	if err != nil {
		return err
	}

	Info(log, obj, path)
	err = vld.Validate(obj.Config())
	if err != nil {
		return err
	}

	return nil
}

func LogValidate(log logger.Logger, vld validator.Validator, obj Object, defaultPath string, optionalPath ...string) error {

	path := utils.OptionalArg(defaultPath, optionalPath...)

	Info(log, obj, path)
	err := vld.Validate(obj.Config())
	if err != nil {
		return err
	}

	return nil
}

func loadConfiguration(cfg config.Config, configPath string, obj Object) (skippedKeys []string, err error) {
	objectValue := reflect.ValueOf(obj.Config())
	return loadValue(cfg, configPath, objectValue)
}

func loadValue(cfg config.Config, configPath string, objectValue reflect.Value) (skippedKeys []string, err error) {

	// Make sure the object is a pointer to a struct.
	if objectValue.Kind() != reflect.Ptr || objectValue.Elem().Kind() != reflect.Struct {
		return nil, errors.New("object must be a pointer to a struct")
	}

	// Get a reflect.Type for the struct.
	objectType := objectValue.Elem().Type()

	// Iterate over the fields of the struct.
	for i := 0; i < objectType.NumField(); i++ {
		// Get the field.
		field := objectType.Field(i)
		key := strings.ToLower(field.Name)

		// Construct the full config path for this field.
		fieldConfigPath := Key(configPath, key)

		// Get the field value.
		fieldValue := objectValue.Elem().Field(i)

		// Set the default value if the field is zero-valued.
		if fieldValue.IsZero() {
			// TODO does not work within arrays
			defaultTag, ok := field.Tag.Lookup("default")
			if ok {
				cfg.SetDefault(fieldConfigPath, defaultTag)
			}
		}

		switch field.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if cfg.IsSet(fieldConfigPath) {
				i := fieldValue.Interface()
				switch i.(type) {
				case time.Duration:
					fieldValue.Set(reflect.ValueOf(cfg.GetDuration(fieldConfigPath)))
				default:
					fieldValue.SetInt(int64(cfg.GetInt(fieldConfigPath)))
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if cfg.IsSet(fieldConfigPath) {
				fieldValue.SetUint(uint64(cfg.GetUint(fieldConfigPath)))
			}
		case reflect.Float64, reflect.Float32:
			if cfg.IsSet(fieldConfigPath) {
				fieldValue.SetFloat(cfg.GetFloat64(fieldConfigPath))
			}
		case reflect.String:
			if cfg.IsSet(fieldConfigPath) {
				fieldValue.SetString(cfg.GetString(fieldConfigPath))
			}
		case reflect.Bool:
			if cfg.IsSet(fieldConfigPath) {
				fieldValue.SetBool(cfg.GetBool(fieldConfigPath))
			}
		case reflect.Slice:
			if cfg.IsSet(fieldConfigPath) {
				if field.Type.Elem().Kind() == reflect.Int {
					slice := cfg.GetIntSlice(fieldConfigPath)
					fieldValue.Set(reflect.ValueOf(slice))
				} else {
					slice := cfg.GetStringSlice(fieldConfigPath)
					fieldValue.Set(reflect.ValueOf(slice))
				}
			}
		case reflect.Struct:
			if field.Anonymous {
				skippedK, err := loadValue(cfg, configPath, fieldValue.Addr())
				if err != nil {
					return nil, err
				}
				if skippedK != nil && len(skippedKeys) > 0 {
					skippedKeys = append(skippedKeys, skippedK...)
				}
			} else {
				i := fieldValue.Interface()
				switch i.(type) {
				case time.Time:
					if cfg.IsSet(fieldConfigPath) {
						fieldValue.Set(reflect.ValueOf(cfg.GetTime(fieldConfigPath)))
					}
				case time.Duration:
					if cfg.IsSet(fieldConfigPath) {
						fieldValue.Set(reflect.ValueOf(cfg.GetDuration(fieldConfigPath)))
					}
				}
			}
		default:
			skippedKeys = append(skippedKeys, key)
		}
	}

	return skippedKeys, nil
}
