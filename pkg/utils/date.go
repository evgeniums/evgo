package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Date int

var DateNil = Date(0)
var TimeNil = time.Time{}

func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")

	if s == "" {
		*d = DateNil
		return nil
	}

	tmp, err := StrToDate(s)
	if err != nil {
		return err
	}
	*d = tmp
	return err
}

func (d *Date) Set(year int, month int, day int) {
	*d = Date(year*10000 + month*100 + day)
}

func (d *Date) SetTime(t time.Time) {
	d.Set(t.Year(), int(t.Month()), t.Day())
}

func (d *Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Date) IsNil() bool {
	return *d == DateNil
}

func (d *Date) Year() int {
	year := int(*d) / 10000
	return year
}

func (d *Date) Month() int {
	month := (int(*d) % 10000) / 100
	return month
}

func (d *Date) Day() int {
	day := int(*d) % 100
	return day
}

func (d *Date) MMonth() Month {
	var month Month
	month.Set(d.Year(), d.Month())
	return month
}

func (d *Date) Time() time.Time {
	return time.Date(d.Year(), time.Month(d.Month()), d.Day(), 0, 0, 0, 0, time.UTC)
}

func (d *Date) String() string {
	if d.IsNil() {
		return ""
	}
	formatted := fmt.Sprintf("%04d-%02d-%02d", d.Year(), d.Month(), d.Day())
	return formatted
}

func (d *Date) StringRu() string {
	formatted := fmt.Sprintf("%02d.%02d.%04d", d.Day(), d.Month(), d.Year())
	return formatted
}

func (d *Date) StringRuShort() string {
	formatted := fmt.Sprintf("%02d.%02d.%02d", d.Day(), d.Month(), d.Year()-2000)
	return formatted
}

func (d *Date) AsNumber() string {
	formatted := fmt.Sprintf("%08d", int(*d))
	return formatted
}

func BeginningOfDay(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func EndOfDay(t time.Time) time.Time {
	b := BeginningOfDay(t)
	r := b.Add(time.Hour * 24).Add(-time.Second)
	return r
}

func StrToDate(s string) (Date, error) {
	if s == "" {
		return DateNil, nil
	}

	iDate, err := StrToUint32(s)
	if err == nil {
		return Date(iDate), nil
	}

	t, err := strDateToTime(s)
	if t == TimeNil {
		return DateNil, err
	}
	var d Date
	d.SetTime(BeginningOfDay(t))
	return d, err
}

func strDateToTime(s string) (time.Time, error) {

	if s == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		t, err = time.Parse("02.01.2006", s)
		if err != nil {
			t, err = time.Parse("02.01.06", s)
			if err != nil {
				return time.Time{}, err
			}
		}
	}
	return t, nil
}

func Today() Date {
	var d Date
	d.SetTime(time.Now())
	return d
}

func Yesterday() Date {
	var d Date
	d.SetTime(time.Now().Add(-time.Hour * 24))
	return d
}

func Tommorrow() Date {
	var d Date
	d.SetTime(time.Now().Add(+time.Hour * 24))
	return d
}

func DateOfTime(t time.Time) Date {
	var d Date
	d.SetTime(t)
	return d
}

func ParseRuTime(str string) (time.Time, error) {

	t, err := time.Parse("02.01.2006 15:04:05", str)
	if err != nil {
		t, err = time.Parse("02.01.2006", str)
		if err != nil {
			t, err = time.Parse("02.01.06 15:04:05", str)
		}
	}

	return t, err
}

func SerializeTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

func ParseTime(str string) (time.Time, error) {

	t, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		t, err = time.Parse("2006-01-02", str)
		if err != nil {
			t, err = time.Parse(time.RFC3339, str)
		}
	}

	return t, err
}

func ParseRuTimeShort(str string) (time.Time, error) {

	t, err := time.Parse("02.01.06 15:04:05", str)
	return t, err
}

func DateConverter(str string) reflect.Value {
	d, _ := StrToDate(str)
	return reflect.ValueOf(d)
}

func DateReflectStr(val reflect.Value) string {
	if !val.CanInt() {
		return ""
	}
	d := Date(val.Int())
	return d.String()
}

func IsTime(obj reflect.Value) bool {
	_, ok := obj.Interface().(time.Time)
	return ok
}

func TimeReflectStr(val reflect.Value) string {

	t, ok := val.Interface().(time.Time)
	if !ok {
		return ""
	}

	return TimeRFC3339(t)
}

func TimeConverter(str string) reflect.Value {
	t, _ := ParseTime(str)
	return reflect.ValueOf(t)
}

func DateFromId(id string) (Date, error) {

	if len(id) != 20 {
		return 0, errors.New("too short ID")
	}

	idTimeStr := id[:8]
	timeInt, err := strconv.ParseInt(idTimeStr, 16, 64)
	if err != nil {
		return 0, errors.New("invalid time in ID")
	}
	tm := time.Unix(timeInt, 0)

	m := DateOfTime(tm)
	return m, nil
}

func TimeInTimezone(tz string, inTime ...time.Time) (time.Time, error) {

	t := OptionalArg(time.Now(), inTime...)

	//init location
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return t, err
	}

	// set timezone
	result := t.In(loc)

	// done
	return result, nil
}

func DateInTimezone(tz string, inTime ...time.Time) (Date, error) {

	t := OptionalArg(time.Now(), inTime...)

	//init location
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return DateOfTime(t), err
	}

	// set timezone
	result := t.In(loc)

	// done
	return DateOfTime(result), nil
}

func DateWithOffset(offset int, inDate ...Date) Date {
	d := OptionalArg(Today(), inDate...)
	if offset == 0 {
		return d
	}

	now := d.Time()
	offsTime := now.Add(time.Hour * time.Duration(offset))
	result := DateOfTime(offsTime)
	return result
}

func ToHatnProtoDatetime(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli() << 8
}

func FromHatnProtoDatetime(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	n := v >> 8
	return time.UnixMilli(n)
}
