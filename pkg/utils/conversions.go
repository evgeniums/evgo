package utils

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"golang.org/x/exp/constraints"
)

type Float interface {
	float32 | float64
}

func StrToFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	str := strings.ReplaceAll(s, ",", ".")
	str = strings.ReplaceAll(str, " ", "")
	f, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return float64(f), nil
}

func FloatToStr[T Float](val T) string {
	str := strconv.FormatFloat(float64(val), 'f', -1, 64)
	return str
}

func FloatToStrComma[T Float](val T) string {
	str := strconv.FormatFloat(float64(val), 'f', -1, 64)
	str = strings.ReplaceAll(str, ".", ",")
	return str
}

func FloatToStr2[T Float](val T) string {
	v := math.Round(float64(val)*100) / 100
	str := strconv.FormatFloat(v, 'f', 2, 64)
	return str
}

func FloatToStr2Comma[T Float](val T) string {
	str := FloatToStr2(val)
	str = strings.ReplaceAll(str, ".", ",")
	return str
}

func FloatToStr2Hyphen[T Float](val T) string {
	str := FloatToStr2(val)
	str = strings.ReplaceAll(str, ".", "-")
	return str
}

func FloatToStr4Comma[T Float](val T) string {
	str := FloatToStr4(val)
	str = strings.ReplaceAll(str, ".", ",")
	return str
}

func FloatToStr4Hyphen[T Float](val T) string {
	str := FloatToStr4(val)
	str = strings.ReplaceAll(str, ".", "-")
	return str
}

func FloatToStr4[T Float](val T) string {
	v := math.Round(float64(val)*10000) / 10000
	str := strconv.FormatFloat(v, 'f', 2, 64)
	return str
}

func MoneyToInteger(dollars float64) int64 {
	return int64(math.Round(float64(dollars) * 100.00))
}

func MoneyToIntegerUp(dollars float64) int64 {
	return int64(math.Ceil(float64(dollars) * 100.00))
}

func MoneyToDecimal[T constraints.Integer](cents T) float64 {
	v := float64(cents) / 100.00
	return float64(v)
}

func FloatToDecimalStr(f float64, comma ...bool) string {
	if OptionalArg(false, comma...) {
		return FloatToStrComma(f)
	}
	return FloatToStr(f)
}

func MoneyToDecimalStr(cents int64, comma ...bool) string {
	m := MoneyToDecimal(cents)
	if OptionalArg(false, comma...) {
		return FloatToStr2Comma(m)
	}
	return FloatToStr2(m)
}

func FloatToStr2Variant[T Float](val T, comma ...bool) string {
	if OptionalArg(false, comma...) {
		return FloatToStr2Comma(val)
	}
	return FloatToStr2(val)
}

func FloatToStr4Variant[T Float](val T, comma ...bool) string {
	if OptionalArg(false, comma...) {
		return FloatToStr4Comma(val)
	}
	return FloatToStr4(val)
}

func RoundMoneyUp(value float64) float64 {
	r := math.Ceil(float64(value)*100) / 100
	return float64(r)
}

func RoundMoneyDown(value float64) float64 {
	r := math.Floor(float64(value)*100) / 100
	return float64(r)
}

func RoundMoney(value float64) float64 {
	r := math.Round(float64(value)*100) / 100
	return r
}

func TimeToStr(t time.Time) string {
	str := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	return str
}

func TimeRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}

func TimeToStrRu(t time.Time) string {
	str := fmt.Sprintf("%02d.%02d.%04d %02d:%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
	return str
}

func StrToUint32(s string) (uint32, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(val), nil
}

func StrToUint64(s string) (uint64, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func StrToInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

func StrToInt32(s string) (int32, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(val), nil
}

func StrToInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func BoolToStr(val bool) string {
	if val {
		return "true"
	}
	return "false"
}

func StrToBool(s string) (bool, error) {
	str := strings.ToLower(s)
	if str == "true" {
		return true, nil
	}
	if str == "false" {
		return true, nil
	}
	return false, errors.New("invalid format")
}

func NumToStr[T constraints.Integer](val T) string {
	return fmt.Sprintf("%d", val)
}

const float64EqualityThreshold = 1e-9

func FloatAlmostEqual[T Float](a, b T) bool {
	return math.Abs(float64(a)-float64(b)) <= float64EqualityThreshold
}

func toTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339, data.(string))
		case reflect.Float64:
			return time.Unix(0, int64(data.(float64))*int64(time.Millisecond)), nil
		case reflect.Int64:
			return time.Unix(0, data.(int64)*int64(time.Millisecond)), nil
		default:
			return data, nil
		}
	}
}

func toDateHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {

		var d Date
		if t != reflect.TypeOf(d) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return StrToDate(data.(string))
		case reflect.Int:
			return Date(data.(int)), nil
		default:
			return data, nil
		}
	}
}

func MapToStruct(in interface{}, out interface{}, tag ...string) error {

	t := OptionalArg("json", tag...)

	// create new map decoder
	meta := &mapstructure.Metadata{}
	config := &mapstructure.DecoderConfig{Metadata: meta, TagName: t, Result: out, Squash: true, WeaklyTypedInput: true, DecodeHook: mapstructure.ComposeDecodeHookFunc(toTimeHookFunc(), toDateHookFunc())}
	dec, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	// fill out object with data from input
	err = dec.Decode(in)
	if err != nil {
		return err
	}

	// done
	return nil
}

func CapitalizeAscii(s string) string {
	if len(s) == 0 {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
