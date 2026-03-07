package config_viper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/viper"
	"github.com/tidwall/jsonc"
)

const (
	MergeDirect string = "direct"
	MergeValue  string = "value"
	MergeArray  string = "array"
)

type ExtendMerge struct {
	Path  string         `json:"path"`
	Rules []IncludeMerge `json:"rules"`
}

type IncludeMerge struct {
	Path string            `json:"path"`
	Mode string            `json:"mode"`
	Map  map[string]string `json:"map"`
}

type ConfigViper struct {
	*viper.Viper
	configFile string
	configType string
}

func New() *ConfigViper {
	v := &ConfigViper{}
	v.Viper = viper.New()
	return v
}

func ReadJsonc(path string) ([]byte, error) {

	// read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("fatal error while reading config file: %s", err)
	}

	// expand relative paths
	dir := filepath.Dir(path)
	dir = strings.ReplaceAll(dir, "\\", "/")
	str := string(data)
	str = strings.ReplaceAll(str, "\"$rel/", fmt.Sprintf("\"%s/", dir))

	// process JSONC
	b := jsonc.ToJSON([]byte(str))

	// done
	return b, nil
}

func ReadConfigFromFile(v *viper.Viper, path string, cfgType string) error {

	v.SetConfigType(cfgType)

	if cfgType != "json" {
		v.SetConfigFile(path)
		err := v.ReadInConfig()
		if err != nil {
			return fmt.Errorf("fatal error while reading config file: %s", err)
		}
	} else {
		b, err := ReadJsonc(path)
		if err != nil {
			return fmt.Errorf("fatal error while reading config file: %s", err)
		}
		r := bytes.NewReader(b)
		if err = v.ReadConfig(r); err != nil {
			msg := fmt.Errorf("failed to parse configuration JSON: %s", err)
			return msg
		}
	}
	return nil
}

func MergeConfigFile(v *viper.Viper, path string, configType ...string) error {

	cfgType := utils.OptionalArg("json", configType...)
	v.SetConfigType(cfgType)

	if cfgType != "json" {
		v.SetConfigFile(path)
		err := v.MergeInConfig()
		if err != nil {
			return fmt.Errorf("fatal error while merging config file: %s", err)
		}
	} else {
		data, err := ReadJsonc(path)
		if err != nil {
			return err
		}
		err = v.MergeConfig(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to merge configuration JSON: %s", err)
		}
	}
	return nil
}

func MakeIncludePath(configFile string, path string) (string, error) {
	r := path
	if !utils.FileExists(r) || !utils.IsFile(r) {
		// try relative path
		r = filepath.Join(filepath.Dir(configFile), r)
		if !utils.FileExists(r) || !utils.IsFile(r) {
			return "", fmt.Errorf("failed to include config file %s or %s", path, r)
		}
	}
	return r, nil
}

func MergeConfigs(fromCfg *ConfigViper, toCfg *ConfigViper, mode string, keysMap map[string]string) error {

	if mode == MergeDirect {
		err := MergeConfigFile(toCfg.Viper, fromCfg.configFile, fromCfg.configType)
		if err != nil {
			return fmt.Errorf("failed to include config file %s to %s: %s", fromCfg.configFile, toCfg.configFile, err)
		}
		return nil
	}

	for from, to := range keysMap {
		if to == "" {
			to = from
		}

		if fromCfg.IsSet(from) {
			if !toCfg.IsSet(to) || mode != MergeArray {
				toCfg.Set(to, fromCfg.Get(from))
			} else {
				oldData := toCfg.Get(to)
				newData := fromCfg.Get(from)

				oldSlice, ok := oldData.([]interface{})
				if !ok {
					return fmt.Errorf("failed to append array from %s to %s: %s is not array in main config", fromCfg.configFile, toCfg.configFile, to)
				}
				newSlice, ok := newData.([]interface{})
				if !ok {
					return fmt.Errorf("failed to append array from %s to %s: %s is not array in included config", fromCfg.configFile, toCfg.configFile, from)
				}
				merge := append(oldSlice, newSlice...)
				toCfg.Set(to, merge)
			}
		}
	}
	return nil
}

func (c *ConfigViper) ConfigFile() string {
	return c.configFile
}

func (c *ConfigViper) ConfigType() string {
	return c.configFile
}

func (c *ConfigViper) Load(fromCfg *ConfigViper) error {

	json := fromCfg.ToString()

	c.Viper = viper.New()
	c.Viper.SetConfigType("json")
	if err := c.Viper.ReadConfig(strings.NewReader(json)); err != nil {
		return fmt.Errorf("failed to re-read viper settings: %s", err)
	}
	return nil
}

func (c *ConfigViper) LoadFile(configFile string, configType ...string) error {

	// setup
	cfgType := utils.OptionalArg("json", configType...)
	c.Viper.SetConfigFile(configFile)
	c.Viper.SetConfigType(cfgType)

	// read main configuration file
	err := ReadConfigFromFile(c.Viper, configFile, cfgType)
	if err != nil {
		return err
	}
	c.configFile = configFile
	c.configType = cfgType

	// check if this config extends other config
	extendKey := "extend"
	if c.Viper.IsSet(extendKey) {

		extend := &ExtendMerge{}
		err = c.Viper.UnmarshalKey(extendKey, extend)
		if err != nil {
			return fmt.Errorf("invalid format of extend section")
		}

		path, err := MakeIncludePath(configFile, extend.Path)
		if err != nil {
			return err
		}

		mainCfg := New()
		err = mainCfg.LoadFile(path, cfgType)
		if err != nil {
			return err
		}

		i := 0
		hasDirect := false
		for _, rule := range extend.Rules {
			err = MergeConfigs(c, mainCfg, rule.Mode, rule.Map)
			if err != nil {
				return err
			}
			i++
			if rule.Mode == MergeDirect {
				hasDirect = true
			}
		}
		if hasDirect && i > 1 {
			return fmt.Errorf("failed to extend config file: rule direct mode overrides all other rules, the direct rule must be exclusive")
		}

		err = c.Load(mainCfg)
		if err != nil {
			return err
		}

		return nil
	}

	// load includes
	includes := c.Viper.GetStringSlice("include")
	for _, include := range includes {

		// TODO support nested includes

		path, err := MakeIncludePath(configFile, include)
		if err != nil {
			return err
		}
		err = MergeConfigFile(c.Viper, path, cfgType)
		if err != nil {
			return fmt.Errorf("failed to include config file %s: %s", path, err)
		}
	}

	// load includes for advanced merging
	mergeSectionKey := "include_advanced"
	if c.Viper.IsSet(mergeSectionKey) {

		mergeSection := c.Viper.Get(mergeSectionKey)
		includeItems, ok := mergeSection.([]interface{})
		if !ok {
			return fmt.Errorf("invalid format of %s section", mergeSectionKey)
		}
		for i := range includeItems {
			item := &IncludeMerge{}
			itemKey := fmt.Sprintf("%s.%d", mergeSectionKey, i)
			err = c.Viper.UnmarshalKey(itemKey, item)
			if err != nil {
				return fmt.Errorf("invalid format of %s", itemKey)
			}

			path, err := MakeIncludePath(configFile, item.Path)
			if err != nil {
				return err
			}

			cfg := New()
			err = cfg.LoadFile(path, cfgType)
			if err != nil {
				return fmt.Errorf("failed to read included configuration %s: %s", item.Path, err)
			}

			err = MergeConfigs(cfg, c, item.Mode, item.Map)
			if err != nil {
				return err
			}
		}

		// reload viper configuration
		err = c.Rebuild()
		if err != nil {
			return err
		}
	}

	// done
	return nil
}

func (c *ConfigViper) LoadString(configStr string, configType ...string) error {

	cfgType := utils.OptionalArg("json", configType...)
	c.Viper.SetConfigType(cfgType)

	err := c.Viper.ReadConfig(strings.NewReader(configStr))
	if err != nil {
		msg := fmt.Errorf("fatal error while reading config string: %s", err)
		return msg
	}

	return nil
}

func (c *ConfigViper) Rebuild() error {
	return c.Load(c)
}

func (c *ConfigViper) ToString() string {
	all := c.AllSettings()
	b, _ := json.MarshalIndent(all, "", "   ")
	return string(b)
}

func (c *ConfigViper) GetFloat64Slice(key string) []float64 {
	if !c.IsSet(key) {
		return []float64{}
	}
	val := c.Get(key)
	iSlice, ok := val.([]interface{})
	if !ok {
		r, ok := val.([]float64)
		if ok {
			return r
		}
		return []float64{}
	}
	l := make([]float64, len(iSlice))
	for i := 0; i < len(iSlice); i++ {
		val, ok := iSlice[i].(float64)
		if !ok {
			return []float64{}
		}
		l[i] = val
	}
	return l
}
