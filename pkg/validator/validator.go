package validator

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/mitchellh/mapstructure"
)

type Validator interface {
	Validate(s interface{}) error
	ValidateValue(value interface{}, rules string) error
	ValidatePartial(s interface{}, fields ...string) *ValidationError
}

type ValidationError struct {
	Message string `json:"message"`
	Field   string `json:"field"`
	Err     error  `json:"error"`
}

func (e *ValidationError) Error() string {
	if e.Message == "" {
		if e.Err == nil {
			return ""
		}
		return e.Err.Error()
	}
	if e.Field != "" {
		return fmt.Sprintf("validation failed on field \"%s\": %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

func (e *ValidationError) GenericError() generic_error.Error {
	err := generic_error.New(generic_error.ErrorCodeFormat, e.Message)
	err.SetDetails(e.Field)
	return err
}

func ValidateMap(v Validator, m map[string]interface{}, sampleStruct interface{}, allowedFields ...string) *ValidationError {

	// collect keys
	keys := utils.AllMapKeys(m)

	// check if there are not allowed keys in the map
	if len(allowedFields) != 0 {
		notAllowedMap := make(map[string]bool)
		for _, field := range allowedFields {
			notAllowedMap[field] = true
		}
		for _, key := range keys {
			_, found := notAllowedMap[key]
			if found {
				err := &ValidationError{}
				err.Message = "map contains not allowed fields"
				err.Field = key
				return err
			}
		}
	}

	// create new decoder
	meta := &mapstructure.Metadata{}
	config := &mapstructure.DecoderConfig{Metadata: meta, TagName: "json", Result: sampleStruct, ErrorUnused: true, Squash: true, WeaklyTypedInput: true}
	dec, err := mapstructure.NewDecoder(config)
	if err != nil {
		panic(fmt.Errorf("failed to create decoder: %s", err))
	}

	// fill struct with data from map
	err = dec.Decode(m)
	if err != nil {
		vErr := &ValidationError{}
		vErr.Message = "Invalid fields for update"
		vErr.Err = err
		return vErr
	}

	// invoke partial validation
	vErr := v.ValidatePartial(sampleStruct, keys...)
	if vErr != nil {
		return vErr
	}

	// done
	return nil
}
