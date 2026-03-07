package db

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
	"github.com/markphelps/optional"
)

type Interval struct {
	From     interface{}
	To       interface{}
	FromOpen bool
	ToOpen   bool
}

type BetweenFields struct {
	FromField string
	ToField   string
	Value     interface{}
	FromOpen  bool
	ToOpen    bool
}

func (i *Interval) IsNull() bool {
	return i.From == nil && i.To == nil
}

type FilterConfig struct {
	SortField     string `json:"sort_field,omitempty"`
	SortDirection string `json:"sort_direction,omitempty" validate:"omitempty,oneof=ASC DESC" vmessage:"Sort direction can be either ASC or DESC"`
	Offset        int    `json:"offset,omitempty" validate:"gte=0" vmessage:"Offset can not be negative"`
	Limit         int    `json:"limit,omitempty" validate:"gte=0" vmessage:"Limit can not be negative"`
	Count         bool   `json:"count,omitempty"`
}

type OrFields struct {
	Value  interface{}
	Fields []string
}

type Filter struct {
	FilterConfig
	Fields        Fields
	FieldsIn      map[string][]interface{}
	FieldsNotIn   map[string][]interface{}
	Intervals     map[string]*Interval
	BetweenFields []*BetweenFields
	OrFields      []*OrFields

	PresetFields []Fields

	NoLimit bool
}

func NewFilter() *Filter {
	f := &Filter{}
	f.PresetFields = make([]Fields, 0)
	return f
}

func (f *Filter) PushPresetFields(fields Fields) {
	f.PresetFields = append(f.PresetFields, fields)
}

func (f *Filter) PopPresetFields() {
	if len(f.PresetFields) > 0 {
		f.PresetFields = f.PresetFields[:len(f.PresetFields)-1]
	}
}

func (f *Filter) SetSorting(field string, direction ...string) {
	f.SortField = field
	f.SortDirection = utils.OptionalArg(SORT_ASC, direction...)
}

func (f *Filter) AddFields(fields Fields) {
	for key, value := range fields {
		f.AddField(key, value)
	}
}

func (f *Filter) AddField(name string, value interface{}) {
	if f.Fields == nil {
		f.Fields = Fields{}
	}
	f.Fields[name] = value
}

func (f *Filter) AddFieldIn(name string, values ...interface{}) {
	if f.FieldsIn == nil {
		f.FieldsIn = make(map[string][]interface{})
	}
	f.FieldsIn[name] = append([]interface{}{}, values...)
}

func (f *Filter) AddFieldNotIn(name string, values ...interface{}) {
	if f.FieldsNotIn == nil {
		f.FieldsNotIn = make(map[string][]interface{})
	}
	f.FieldsNotIn[name] = append([]interface{}{}, values...)
}

func (f *Filter) AddInterval(name string, from interface{}, to interface{}) {
	if f.Intervals == nil {
		f.Intervals = make(map[string]*Interval)
	}
	f.Intervals[name] = &Interval{From: from, To: to}
}

func (f *Filter) AddBetweenField(fromField string, toField string, value interface{}) {
	if f.BetweenFields == nil {
		f.BetweenFields = make([]*BetweenFields, 0, 1)
	}
	f.BetweenFields = append(f.BetweenFields, &BetweenFields{Value: value, FromField: fromField, ToField: toField})
}

func (f *Filter) AddOrFields(value interface{}, names ...string) {
	if f.OrFields == nil {
		f.OrFields = make([]*OrFields, 0)
	}
	item := &OrFields{}
	item.Value = value
	item.Fields = append([]string{}, names...)
	f.OrFields = append(f.OrFields, item)
}

func filterValueToString(value interface{}) string {
	return fmt.Sprintf("%v", value)
}

func (f *Filter) ToQuery() *Query {
	q := &Query{}

	q.FilterConfig = f.FilterConfig

	// fill fields
	if len(f.Fields) > 0 {
		q.Fields = make(map[string]string)
	}
	for key, value := range f.Fields {
		q.Fields[key] = filterValueToString(value)
	}

	// fill fields_in
	if len(f.FieldsIn) > 0 {
		q.FieldsIn = make(map[string][]string)
	}
	for key, values := range f.FieldsIn {
		arr := make([]string, len(values))
		for i := 0; i < len(values); i++ {
			arr[i] = filterValueToString(values[i])
		}
		q.FieldsIn[key] = arr
	}

	// fill fields_not_in
	if len(f.FieldsNotIn) > 0 {
		q.FieldsNotIn = make(map[string][]string)
	}
	for key, values := range f.FieldsNotIn {
		arr := make([]string, len(values))
		for i := 0; i < len(values); i++ {
			arr[i] = filterValueToString(values[i])
		}
		q.FieldsNotIn[key] = arr
	}

	// fill intervals
	if len(f.Intervals) > 0 {
		q.Intervals = make(map[string]QueryInterval)
	}
	for key, interval := range f.Intervals {
		to := optional.String{}
		from := optional.String{}
		if interval.From != nil {
			from.Set(filterValueToString(interval.From))
		}
		if interval.To != nil {
			to.Set(filterValueToString(interval.To))
		}
		q.Intervals[key] = QueryInterval{From: from, To: to, FromOpen: interval.FromOpen, ToOpen: interval.ToOpen}
	}

	// fill betweens
	if len(f.BetweenFields) > 0 {
		q.BetweenFields = make([]QueryBetweenFields, len(f.BetweenFields))
	}
	for i := 0; i < len(f.BetweenFields); i++ {
		betweenQ := f.BetweenFields[i]
		q.BetweenFields[i] = QueryBetweenFields{FromField: betweenQ.FromField, ToField: betweenQ.ToField, Value: filterValueToString(betweenQ.Value), FromOpen: betweenQ.FromOpen, ToOpen: betweenQ.ToOpen}
	}

	// fill or_fields
	if len(f.OrFields) > 0 {
		q.OrFields = make([]QueryOrFields, len(f.OrFields))
	}
	for i := 0; i < len(f.OrFields); i++ {
		orFields := f.OrFields[i]
		value := filterValueToString(orFields.Value)
		q.OrFields[i] = QueryOrFields{Value: value, Fields: orFields.Fields}
	}

	return q
}

func (f *Filter) ToQueryString() string {
	q := f.ToQuery()
	b, _ := json.Marshal(q)
	return string(b)
}

type QueryInterval struct {
	From     optional.String `json:"from,omitempty"`
	To       optional.String `json:"to,omitempty"`
	FromOpen bool            `json:"from_open,omitempty"`
	ToOpen   bool            `json:"to_open,omitempty"`
}

type QueryBetweenFields struct {
	FromField string `json:"from_field,omitempty"`
	ToField   string `json:"to_field,omitempty"`
	Value     string `json:"value"`
	FromOpen  bool   `json:"from_open,omitempty"`
	ToOpen    bool   `json:"to_open,omitempty"`
}

type QueryOrFields struct {
	Value  string   `json:"value"`
	Fields []string `json:"fields"`
}

type Query struct {
	FilterConfig

	Fields        map[string]string        `json:"fields,omitempty"`
	FieldsIn      map[string][]string      `json:"fields_in,omitempty"`
	FieldsNotIn   map[string][]string      `json:"fields_not_in,omitempty"`
	Intervals     map[string]QueryInterval `json:"intervals,omitempty"`
	BetweenFields []QueryBetweenFields     `json:"betwees_fields,omitempty"`
	OrFields      []QueryOrFields          `json:"or_fields,omitempty"`
}

type WithFilterParser interface {
	PrepareFilterParser(model interface{}, name string, validator ...*FilterValidator) (FilterParser, error)
	ParseFilter(query *Query, parserName string) (*Filter, error)
	ParseFilterDirect(query *Query, model interface{}, name string, validator ...*FilterValidator) (*Filter, error)
}

type FilterParser interface {
	Parse(query *Query) (*Filter, error)
}

type FilterValidator struct {
	Validator       validator.Validator
	Rules           map[string]string
	PlainFieldNames bool
}

func ParseQuery(db DB, query string, model interface{}, parserName string, validator ...*FilterValidator) (*Filter, error) {

	if query == "" {
		return nil, nil
	}

	q := &Query{}

	dec := json.NewDecoder(strings.NewReader(query))
	dec.DisallowUnknownFields()
	for {
		if err := dec.Decode(q); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return db.ParseFilterDirect(q, model, parserName, validator...)
}

func EmptyFilterValidator(vld validator.Validator) *FilterValidator {
	return &FilterValidator{Validator: vld}
}
