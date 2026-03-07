package db_gorm

import (
	"errors"
	"fmt"
	"sync"

	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
	"github.com/markphelps/optional"
	"gorm.io/gorm/schema"
)

type FilterParser struct {
	Manager     *FilterManager
	Destination *ModelDescriptor
	Validator   *db.FilterValidator
}

func convertValue(field *schema.Field, value string) (interface{}, error) {

	switch field.DataType {
	case schema.String:
		return value, nil
	case schema.Int:
		val, err := utils.StrToInt64(value)
		if err != nil {
			return nil, err
		}
		return val, nil
	case schema.Bool:
		val, err := utils.StrToBool(value)
		if err != nil {
			return nil, err
		}
		return val, nil
	case schema.Uint:
		val, err := utils.StrToUint64(value)
		if err != nil {
			return nil, err
		}
		return val, nil
	case schema.Float:
		val, err := utils.StrToFloat(value)
		if err != nil {
			return nil, err
		}
		return val, nil
	case schema.Time:
		val, err := utils.ParseTime(value)
		if err != nil {
			return nil, err
		}
		return val, nil
	}

	return nil, errors.New("unsupported field type")
}

func (f *FilterParser) ParseValidateField(name string, value string, onlyName ...bool) (string, interface{}, error) {

	fieldName := func(dscr *FieldDescriptor) string {
		if f.Validator != nil && f.Validator.PlainFieldNames {
			return dscr.DbField
		}
		return dscr.FullDbName
	}

	// find field by json name
	field, err := f.Destination.FindJsonField(name)
	if err != nil {
		return "", nil, &validator.ValidationError{Message: "Invalid field name", Field: name}
	}

	// allow only indexed fields
	if !field.Index {
		return "", nil, &validator.ValidationError{Message: "Non-index field can not be used in queries", Field: name}
	}

	// break if only field name validation required
	if utils.OptionalArg(false, onlyName...) {
		return fieldName(field), value, nil
	}

	// convert string value to desired type
	result, err := convertValue(field.Schema, value)
	if err != nil {
		return "", nil, &validator.ValidationError{Message: "Invalid field value", Field: name}
	}

	// validate result
	if f.Validator != nil {
		useExplicitRules := false
		fieldRules := ""
		if f.Validator.Rules != nil {
			fieldRules, useExplicitRules = f.Validator.Rules[name]
		}
		if !useExplicitRules {
			// use tag rules
			fieldRules = field.Schema.Tag.Get("validate")
		}
		if fieldRules != "" {
			err = f.Validator.Validator.ValidateValue(result, fieldRules)
			if err != nil {
				return "", nil, err
			}
		}
	}

	// done
	return fieldName(field), result, nil
}

func (f *FilterParser) Parse(query *db.Query) (*db.Filter, error) {

	// setup
	var err error
	if query == nil {
		return nil, nil
	}
	filter := &db.Filter{}
	filter.FilterConfig = query.FilterConfig
	if f.Validator != nil {
		err = f.Validator.Validator.Validate(filter.FilterConfig)
		if err != nil {
			return nil, err
		}
	}

	// sort field
	if query.SortField != "" {
		field, _, err := f.ParseValidateField(query.SortField, "", true)
		if err != nil {
			return nil, err
		}
		filter.SortField = field
	}

	// fill fields
	if len(query.Fields) > 0 {
		filter.Fields = make(map[string]interface{})
	}
	for key, value := range query.Fields {
		field, value, err := f.ParseValidateField(key, value)
		if err != nil {
			return nil, err
		}
		filter.Fields[field] = value
	}

	// fill fields_in
	if len(query.FieldsIn) > 0 {
		filter.FieldsIn = make(map[string][]interface{})
	}
	for key, values := range query.FieldsIn {
		arr := make([]interface{}, len(values))
		field := key
		for i := 0; i < len(values); i++ {
			field, arr[i], err = f.ParseValidateField(key, values[i])
			if err != nil {
				return nil, err
			}
		}
		filter.FieldsIn[field] = arr
	}

	// fill fields_not_in
	if len(query.FieldsNotIn) > 0 {
		filter.FieldsNotIn = make(map[string][]interface{})
	}
	for key, values := range query.FieldsNotIn {
		arr := make([]interface{}, len(values))
		field := key
		for i := 0; i < len(values); i++ {
			field, arr[i], err = f.ParseValidateField(key, values[i])
			if err != nil {
				return nil, err
			}
		}
		filter.FieldsNotIn[field] = arr
	}

	extractOptional := func(field string, opt optional.String) (string, bool, error) {
		if !opt.Present() {
			return "", false, nil
		}
		str, err := opt.Get()
		if err != nil {
			return "", true, &validator.ValidationError{Message: "Invalid field type", Field: field}
		}
		return str, true, nil
	}

	// fill intervals
	if len(query.Intervals) > 0 {
		filter.Intervals = make(map[string]*Interval)
	}
	for key, interval := range query.Intervals {
		var from interface{}
		var to interface{}
		var field string

		fromStr, present, err := extractOptional("interval.from", interval.From)
		if err != nil {
			return nil, err
		}
		if present {
			field, from, err = f.ParseValidateField(key, fromStr)
			if err != nil {
				return nil, err
			}
		}

		toStr, present, err := extractOptional("interval.to", interval.To)
		if err != nil {
			return nil, err
		}
		if present {
			var fieldTo string
			fieldTo, to, err = f.ParseValidateField(key, toStr)
			if err != nil {
				return nil, err
			}
			if field == "" {
				field = fieldTo
			}
		}

		if field != "" {
			filter.Intervals[field] = &Interval{From: from, To: to, FromOpen: interval.FromOpen, ToOpen: interval.ToOpen}
		}
	}

	// fill betweens
	if len(query.BetweenFields) > 0 {
		filter.BetweenFields = make([]*db.BetweenFields, len(query.BetweenFields))
	}
	for i := 0; i < len(query.BetweenFields); i++ {
		betweenQ := query.BetweenFields[i]
		fromField, val, err := f.ParseValidateField(betweenQ.FromField, betweenQ.Value)
		if err != nil {
			return nil, err
		}
		toField, _, err := f.ParseValidateField(betweenQ.ToField, betweenQ.Value)
		if err != nil {
			return nil, err
		}
		filter.BetweenFields[i] = &db.BetweenFields{FromField: fromField, ToField: toField, Value: val, FromOpen: betweenQ.FromOpen, ToOpen: betweenQ.ToOpen}
	}

	// fill or_fields
	if len(query.OrFields) > 0 {
		filter.OrFields = make([]*db.OrFields, len(query.OrFields))
	}
	for i := 0; i < len(query.OrFields); i++ {
		orFieldsQ := query.OrFields[i]

		var value interface{}
		fields := make([]string, len(orFieldsQ.Fields))
		for j := 0; j < len(orFieldsQ.Fields); j++ {
			field, val, err := f.ParseValidateField(orFieldsQ.Fields[j], orFieldsQ.Value)
			if err != nil {
				return nil, err
			}
			fields[j] = field
			value = val
		}

		filter.OrFields[i] = &db.OrFields{Value: value, Fields: fields}
	}

	// done
	return filter, nil
}

type FilterManager struct {
	mutex      sync.Mutex
	modelStore *ModelStore
	parsers    map[string]*FilterParser
}

func NewFilterManager(modelStore ...*ModelStore) *FilterManager {
	f := &FilterManager{}
	f.modelStore = utils.OptionalArg(GlobalModelStore, modelStore...)
	f.parsers = make(map[string]*FilterParser)
	return f
}

func (f *FilterManager) PrepareFilterParser(model interface{}, name string, validator ...*db.FilterValidator) (db.FilterParser, error) {

	parser := &FilterParser{}
	parser.Manager = f

	// parse schema
	s, err := schema.Parse(model, f.modelStore.schemaCache, f.modelStore.schemaNamer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model schema: %s", err)
	}
	descriptor := f.modelStore.FindDescriptor(s.Table)
	if descriptor == nil {
		return nil, fmt.Errorf("model %s (table %s) not registered", s.Name, s.Table)
	}
	if !descriptor.FieldsReady() {
		err = f.modelStore.ParseModelFields(descriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fields for model %s (table %s): %s", s.Name, s.Table, err)
		}
	}
	parser.Destination = descriptor

	// keep validator
	parser.Validator = utils.OptionalArg(nil, validator...)

	// save parser in cache
	f.mutex.Lock()
	f.parsers[name] = parser
	f.mutex.Unlock()

	// done
	return parser, nil
}

func (f *FilterManager) ParseFilter(query *db.Query, parserName string) (*db.Filter, error) {
	f.mutex.Lock()
	parser, ok := f.parsers[parserName]
	f.mutex.Unlock()
	if !ok {
		return nil, &validator.ValidationError{Message: "unknown parser"}
	}
	return parser.Parse(query)
}

func (f *FilterManager) ParseFilterDirect(query *db.Query, model interface{}, parserName string, vld ...*db.FilterValidator) (*db.Filter, error) {

	f.mutex.Lock()
	parser, ok := f.parsers[parserName]
	f.mutex.Unlock()
	if ok {
		return parser.Parse(query)
	}

	p, err := f.PrepareFilterParser(model, parserName, vld...)
	if err != nil {
		return nil, err
	}

	return p.Parse(query)
}
