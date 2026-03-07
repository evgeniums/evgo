package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/evgeniums/evgo/pkg/utils"
)

const (
	TypeEmpty      = ""
	TypeBool       = "bool"
	TypeInt        = "int"
	TypeIntList    = "int_list"
	TypeFloat      = "float"
	TypeFloatList  = "float_list"
	TypeString     = "string"
	TypeStringList = "string_list"
)

var knownArgTypes = map[string]bool{
	TypeEmpty:      true,
	TypeBool:       true,
	TypeInt:        true,
	TypeFloat:      true,
	TypeString:     true,
	TypeIntList:    true,
	TypeFloatList:  true,
	TypeStringList: true,
}

func setParameter(cfg Config, valueType string, key string, value string) error {

	switch valueType {
	case TypeEmpty:
		cfg.Set(key, value)
	case TypeString:
		cfg.Set(key, value)
	case TypeBool:
		val := strings.ToLower(value)
		cfg.Set(key, val == "true")
	case TypeInt:
		val, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parameter %s must be integer", key)
		}
		cfg.Set(key, val)
	case TypeFloat:
		val, err := utils.StrToFloat(value)
		if err != nil {
			return fmt.Errorf("parameter %s must be float", key)
		}
		cfg.Set(key, val)
	case TypeStringList:
		vals := strings.Split(value, ",")
		cfg.Set(key, vals)
	case TypeIntList:
		{
			strVals := strings.Split(value, ",")
			vals := make([]int, len(strVals))
			for i, strVal := range strVals {
				val, err := strconv.Atoi(strVal)
				if err != nil {
					return fmt.Errorf("item %d of parameter %s must be integer", i, key)
				}
				vals[i] = val
			}
			cfg.Set(key, vals)
		}
	case TypeFloatList:
		{
			strVals := strings.Split(value, ",")
			vals := make([]float64, len(strVals))
			for i, strVal := range strVals {
				val, err := utils.StrToFloat(strVal)
				if err != nil {
					return fmt.Errorf("item %d of parameter %s must be float", i, key)
				}
				vals[i] = val
			}
			cfg.Set(key, vals)
		}
	}

	return nil
}

func LoadArgs(cfg Config, args []string) error {

	if len(args) == 0 {
		return nil
	}

	if len(args)%2 != 0 {
		return fmt.Errorf("invalid number of configuration parameters in arguments: %d (must be even)", len(args))
	}

	i := 0
	for i < len(args) {

		key := args[i]
		i++
		value := args[i]
		i++

		if !strings.HasPrefix(key, "--") {
			return fmt.Errorf("name of configuration parameter (%s) must start with --", key)
		}

		key = key[2:]
		key = strings.ToLower(key)
		keyParts := strings.Split(key, ".")
		if len(keyParts) < 2 {
			return fmt.Errorf("last section of configuration parameter name (%s) must denote the parameter's type", key)
		}

		parameterType := keyParts[len(keyParts)-1]
		_, ok := knownArgTypes[parameterType]
		if !ok {
			return fmt.Errorf("unknown type %s of configuration parameter %s", parameterType, key)
		}

		keyParts = keyParts[:len(keyParts)-1]
		key = strings.Join(keyParts, ".")

		err := setParameter(cfg, parameterType, key, value)
		if err != nil {
			return err
		}
	}

	cfg.Rebuild()
	return nil
}
