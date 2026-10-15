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
	if t.IsZero() {
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

const (
	// epochBit is the 8th bit (0x80). It marks the Unix epoch instant (0 ms, UTC), whose packed
	// form would otherwise be 0, which means "not set".
	epochBit = 1 << 7
	tzMask   = 0x7F
)

// PackDatetime compresses time into int64: [millis << 8][7-bit tz in 15-minute units].
//
// Same rule as hatn DateTimePacker (hatn/common/datetime.h): 0 is "not set"; the epoch bit is set
// only for the epoch instant in UTC so that it does not collide with "not set".
func PackDatetime(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}

	milli := t.UnixMilli()

	// timezone offset in 15-min increments
	_, offset := t.Zone()
	tzValue := int64(offset / 60 / 15)

	var epoch int64 = 0
	if milli == 0 && tzValue == 0 {
		epoch = epochBit
	}

	return (milli << 8) | epoch | (tzValue & tzMask)
}

// UnpackDatetime extracts the time from the packed int64.
//
// 0 means "not set" and yields the zero time; any other value is millis = packed>>8 with a
// 7-bit signed timezone in the low bits. Clients (hatn DateTimePacker) never set the epoch bit
// for ordinary timestamps, so it must not be required.
func UnpackDatetime(packed int64) time.Time {
	if packed == 0 {
		return time.Time{}
	}

	milli := packed >> 8
	tzValue := packed & tzMask

	// Sign extend 7-bit timezone for negative offsets
	if tzValue > 63 {
		tzValue -= 128
	}

	offsetSeconds := int(tzValue * 15 * 60)
	// Using "" for name to avoid side effects in string output
	loc := time.FixedZone("", offsetSeconds)

	return time.UnixMilli(milli).In(loc)
}

func ToHatnProtoDatetime(t time.Time) int64 {
	return PackDatetime(t)
}

func FromHatnProtoDatetime(n int64) time.Time {
	return UnpackDatetime(n).Local()
}

func IsExpired(dt *time.Time) bool {
	if dt == nil {
		return false
	}
	if dt.IsZero() {
		return false
	}
	return time.Now().After(*dt)
}
