package config_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/evgeniums/evgo/pkg/config/config_viper"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger/logger_logrus"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator/validator_playground"
	"gopkg.in/go-playground/assert.v1"
)

type config0 struct {
	FIELD_TIME_DURATION time.Duration
}
type sample0 struct {
	config0
}

func (s *sample0) Config() interface{} {
	return &s.config0
}

type embeddedConfig1 struct {
	FIELD_EMBEDDED_STRING string `default:"default_value1"`
}

type config1 struct {
	embeddedConfig1
	FIELD_INT          int
	FIELD_INT8         int8
	FIELD_INT16        int16
	FIELD_INT32        int32
	FIELD_INT64        int64
	FIELD_UINT         uint
	FIELD_UINT8        uint8
	FIELD_UINT16       uint16
	FIELD_UINT32       uint32 `default:"200"`
	FIELD_UINT64       uint64
	FIELD_BOOL         bool `default:"true"`
	FIELD_FLOAT32      float32
	FIELD_FLOAT64      float64
	FIELD_SLICE_INT    []int
	FIELD_SLICE_STRING []string

	FIELD_TIME          time.Time
	FIELD_TIME_DURATION time.Duration
}

type sample1 struct {
	config1
}

func (s *sample1) Config() interface{} {
	return &s.config1
}

func TestObjectConfig(t *testing.T) {

	sampleConfig1 := `{"field_embedded_string":"value1","field_int":100, "field_time_duration": "1h"}`
	cfg1 := config_viper.New()
	err := cfg1.LoadString(sampleConfig1)
	if err != nil {
		t.Fatalf("failed to load configuration from string: %s", err)
	}

	s0 := &sample0{}
	err = object_config.Load(cfg1, s0, "")
	if err != nil {
		t.Fatalf("failed to load object configuration: %s", err)
	}

	s1 := &sample1{}
	err = object_config.Load(cfg1, s1, "")
	if err != nil {
		t.Fatalf("failed to load object configuration: %s", err)
	}

	if s1.FIELD_EMBEDDED_STRING != "value1" {
		t.Errorf("invalid field_embedded_string: expected %s, got %s", "value1", s1.FIELD_EMBEDDED_STRING)
	}
	if s1.FIELD_INT != 100 {
		t.Errorf("invalid field_int: expected %d, got %d", 100, s1.FIELD_INT)
	}
	if s1.FIELD_UINT32 != 200 {
		t.Errorf("invalid field_uint32: expected %d, got %d", 200, s1.FIELD_INT)
	}
	if !s1.FIELD_BOOL {
		t.Errorf("invalid field_bool: expected %v, got %v", true, s1.FIELD_BOOL)
	}

	sampleConfig2 := `
	{
		"field_int":100,
		"field_int8":101,
		"field_int16":-102,
		"field_int32":-103,
		"field_int64":104,
		"field_uint":200,
		"field_uint8":201,
		"field_uint16":202,
		"field_uint32":203,
		"field_uint64":204,
		"field_bool":false,
		"field_float32":1000.01,
		"field_float64":2000.02,
		"field_slice_int":[1,2,3,4],
		"field_slice_string":["a","b","c","d"],
		"field_time": "2024-02-22",
		"field_time_duration": "1h"
	}
`
	cfg2 := config_viper.New()
	err = cfg2.LoadString(sampleConfig2)
	if err != nil {
		t.Fatalf("failed to load configuration 2 from string: %s", err)
	}
	s2 := &sample1{}
	err = object_config.Load(cfg2, s2, "")
	if err != nil {
		t.Fatalf("failed to load object 2 configuration: %s", err)
	}

	if s2.FIELD_EMBEDDED_STRING != "default_value1" {
		t.Errorf("invalid field_embedded_string: expected %s, got %s", "default_value1", s1.FIELD_EMBEDDED_STRING)
	}
	if s2.FIELD_INT != 100 {
		t.Errorf("invalid field_int: expected %d, got %d", 100, s2.FIELD_INT)
	}
	if s2.FIELD_INT8 != 101 {
		t.Errorf("invalid field_int8: expected %d, got %d", 101, s2.FIELD_INT8)
	}
	if s2.FIELD_INT16 != -102 {
		t.Errorf("invalid field_int16: expected %d, got %d", -102, s2.FIELD_INT)
	}
	if s2.FIELD_INT32 != -103 {
		t.Errorf("invalid field_int32: expected %d, got %d", -103, s2.FIELD_INT)
	}
	if s2.FIELD_INT64 != 104 {
		t.Errorf("invalid field_int64: expected %d, got %d", 104, s2.FIELD_INT)
	}

	if s2.FIELD_UINT != 200 {
		t.Errorf("invalid field_int: expected %d, got %d", 200, s2.FIELD_INT)
	}
	if s2.FIELD_UINT8 != 201 {
		t.Errorf("invalid field_uint8: expected %d, got %d", 201, s2.FIELD_INT)
	}
	if s2.FIELD_UINT16 != 202 {
		t.Errorf("invalid field_uint16: expected %d, got %d", 202, s2.FIELD_INT)
	}
	if s2.FIELD_UINT32 != 203 {
		t.Errorf("invalid field_uint32: expected %d, got %d", 203, s2.FIELD_INT)
	}
	if s2.FIELD_UINT64 != 204 {
		t.Errorf("invalid field_uint64: expected %d, got %d", 204, s2.FIELD_INT)
	}

	if s2.FIELD_BOOL {
		t.Errorf("invalid field_bool: expected %v, got %v", false, s2.FIELD_BOOL)
	}
	if !utils.FloatAlmostEqual(s2.FIELD_FLOAT32, 1000.01) {
		t.Errorf("invalid field_float32: expected %s, got %s", utils.FloatToStr2(1000.01), utils.FloatToStr2(s2.FIELD_FLOAT32))
	}
	if !utils.FloatAlmostEqual(s2.FIELD_FLOAT64, 2000.02) {
		t.Errorf("invalid field_float64: expected %s, got %s", utils.FloatToStr2(2000.02), utils.FloatToStr2(s2.FIELD_FLOAT64))
	}

	if !reflect.DeepEqual(s2.FIELD_SLICE_INT, []int{1, 2, 3, 4}) {
		t.Errorf("invalid field_slice_int: expected %v, got %v", []int{1, 2, 3, 4}, s2.FIELD_BOOL)
	}

	if !reflect.DeepEqual(s2.FIELD_SLICE_STRING, []string{"a", "b", "c", "d"}) {
		t.Errorf("invalid field_slice_string: expected %v, got %v", []string{"a", "b", "c", "d"}, s2.FIELD_SLICE_STRING)
	}

	assert.Equal(t, 1, int(s2.FIELD_TIME_DURATION.Hours()))
	tm, _ := utils.ParseTime("2024-02-22")
	assert.Equal(t, tm, s2.FIELD_TIME)

	s3 := &sample1{}

	log := logger_logrus.New()
	vld := validator_playground.New()
	err = object_config.LoadLogValidate(cfg2, log, vld, s3, "")
	if err != nil {
		t.Fatalf("Failed to load s3 configuration: %s", err)
	}
}
