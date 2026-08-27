package validator_playground

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/validator"
	playground "github.com/go-playground/validator/v10"
)

type PlaygroundValdator struct {
	validator *playground.Validate
}

func New() *PlaygroundValdator {
	p := &PlaygroundValdator{validator: playground.New()}
	p.validator.RegisterValidation("alphanum_", ValidateAlphanumUnderscore)
	p.validator.RegisterValidation("phone", ValidatePhone)
	p.validator.RegisterValidation("id", ValidateId)
	p.validator.RegisterValidation("user", ValidateUser)
	p.validator.RegisterValidation("base58", ValidateBase58)

	return p
}

func (v *PlaygroundValdator) Validate(s interface{}) error {
	err := v.validator.Struct(s)
	if err != nil {
		field, msg, err := v.doValidation(s)
		return &validator.ValidationError{Field: field, Message: msg, Err: err}
	}

	return nil
}

func (v *PlaygroundValdator) ValidateValue(value interface{}, rules string) error {
	err := v.validator.Var(value, rules)
	if err != nil {
		return &validator.ValidationError{Field: "value", Err: err}
	}
	return nil
}

func (v *PlaygroundValdator) ValidatePartial(s interface{}, fields ...string) *validator.ValidationError {
	err := v.validator.StructPartial(s, fields...)
	if err != nil {
		field, msg, err := v.doValidation(s, fields...)
		return &validator.ValidationError{Field: field, Message: msg, Err: err}
	}

	return nil
}

// concreteStructValue resolves value through any number of pointer/interface indirections down to
// the underlying struct's type and value. go-playground validates a nested struct found behind an
// interface{}/any-typed field using that value's DYNAMIC type (it doesn't require "dive" for a
// plain struct field, only for slice/map/array elements) -- so resolving a validation error's
// field path back to a *reflect.StructField afterwards must follow the same dynamic type, not the
// field's statically declared type, which is `interface{}` itself (Kind() == Interface) and
// therefore not something reflect.Type.FieldByName can be called on at all.
func concreteStructValue(value reflect.Value) (reflect.Value, bool) {
	for {
		switch value.Kind() {
		case reflect.Ptr, reflect.Interface:
			if value.IsNil() {
				return reflect.Value{}, false
			}
			value = value.Elem()
		case reflect.Struct:
			return value, true
		default:
			return reflect.Value{}, false
		}
	}
}

func (v *PlaygroundValdator) validationSubfield(parentValue reflect.Value, typenames []string) (reflect.StructField, bool) {

	if len(typenames) == 0 {
		return reflect.StructField{}, false
	}
	first := typenames[0]

	structValue, ok := concreteStructValue(parentValue)
	if !ok {
		return reflect.StructField{}, false
	}

	field, ok := structValue.Type().FieldByName(first)
	if !ok {
		return reflect.StructField{}, false
	}

	if len(typenames) == 1 {
		return field, true
	}

	return v.validationSubfield(structValue.FieldByName(first), typenames[1:])
}

func (v *PlaygroundValdator) doValidation(s interface{}, fields ...string) (string, string, error) {
	var err error
	if len(fields) == 0 {
		err = v.validator.Struct(s)
	} else {
		err = v.validator.StructPartial(s, fields...)
	}
	if err != nil {
		var name, message string
		errs := err.(playground.ValidationErrors)
		if len(errs) > 0 {
			fieldErr := errs[0]
			sv, ok := concreteStructValue(reflect.ValueOf(s))
			if !ok {
				return fieldErr.Field(), "", err
			}
			t := sv.Type()

			names := strings.Split(fieldErr.StructNamespace(), ".")
			f1, found := t.FieldByName(names[1])
			if !found {
				return fieldErr.Field(), "", err
			}
			var f reflect.StructField
			if len(names) > 2 {
				f, found = v.validationSubfield(sv.FieldByName(names[1]), names[2:])
				if !found {
					return fieldErr.Field(), "", err
				}
			} else {
				f = f1
			}

			name = f.Name
			tag, _ := f.Tag.Lookup("json")
			if tag == "" {
				tag, _ = f.Tag.Lookup("config")
			}
			if tag != "" {
				name = tag
			}

			message, _ = f.Tag.Lookup("vmessage")
		}
		return name, message, err
	}
	return "", "", nil
}

const alphaNumericUnderscoreRegexString = "^[a-zA-Z0-9_]+$"

var alphaNumericUnerscoreRegex = regexp.MustCompile(alphaNumericUnderscoreRegexString)

func ValidateAlphanumUnderscore(fl playground.FieldLevel) bool {
	return alphaNumericUnerscoreRegex.MatchString(fl.Field().String())
}

const phoneRegexString = "^[1-9]?[0-9]{7,14}$"

var phoneRegex = regexp.MustCompile(phoneRegexString)

func ValidatePhone(fl playground.FieldLevel) bool {
	return phoneRegex.MatchString(fl.Field().String())
}

func ValidateId(fl playground.FieldLevel) bool {
	return common.ValidateId(fl.Field().String())
}

const UserRegExp = "^[a-z_][a-z0-9_\\-]*$"

var UserRegex = regexp.MustCompile(UserRegExp)

func ValidateUser(fl playground.FieldLevel) bool {
	return UserRegex.MatchString(fl.Field().String())
}

var base58Regexp = regexp.MustCompile("^[1-9A-HJ-NP-Za-km-z]+$")

func ValidateBase58(fl playground.FieldLevel) bool {
	field := fl.Field().String()
	if field == "" {
		return true // omitempty will handle empty checks
	}
	return base58Regexp.MatchString(field)
}
