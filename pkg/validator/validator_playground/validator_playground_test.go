package validator_playground

import (
	"strings"
	"testing"

	"github.com/evgeniums/evgo/pkg/validator"
)

// TestValidateAnyFieldHoldingTaggedStruct reproduces the crash found in
// whitemservergo/todos/bug-validator-playground-any-field-panic.md: go-playground's Struct()
// recurses into a struct value found behind an interface{}/any-typed field automatically (no
// "dive" needed -- that's only for slice/map/array elements). When that nested struct's own tag
// fails, resolving the failing field's name/message used to panic
// ("reflect: FieldByName of non-struct type interface {}") instead of returning a normal
// *validator.ValidationError, because validationSubfield looked at the outer field's statically
// declared type (interface{}) rather than the concrete value's dynamic type.
func TestValidateAnyFieldHoldingTaggedStruct(t *testing.T) {

	type namedInner struct {
		Value string `json:"value" validate:"required,max=3" vmessage:"Too long value"`
	}

	type outer struct {
		Data any `json:"data"`
	}

	v := New()
	err := v.Validate(&outer{Data: &namedInner{Value: "way too long"}})
	if err == nil {
		t.Fatal("expected a validation error for the oversize nested field")
	}
	vErr, ok := err.(*validator.ValidationError)
	if !ok {
		t.Fatalf("expected *validator.ValidationError, got %T: %v", err, err)
	}
	if vErr.Field != "value" {
		t.Errorf("expected resolved field name %q, got %q", "value", vErr.Field)
	}
	if !strings.Contains(vErr.Message, "Too long") {
		t.Errorf("expected vmessage to surface, got %q", vErr.Message)
	}
}

// TestValidateAnyFieldNilNeverCrashes guards the edge concreteStructValue must handle without
// panicking: a nil interface behind the any field, with the actual validation failure on a
// sibling field (namespace depth 2, so validationSubfield's recursion is never entered at all).
func TestValidateAnyFieldNilNeverCrashes(t *testing.T) {

	type outer struct {
		Required string `json:"required" validate:"required"`
		Data     any    `json:"data"`
	}

	v := New()
	err := v.Validate(&outer{Data: nil})
	if err == nil {
		t.Fatal("expected a validation error for the missing required field")
	}
	vErr, ok := err.(*validator.ValidationError)
	if !ok {
		t.Fatalf("expected *validator.ValidationError, got %T: %v", err, err)
	}
	if vErr.Field != "required" {
		t.Errorf("expected resolved field name %q, got %q", "required", vErr.Field)
	}
}

// TestValidateAnyFieldDeeplyNested confirms the fix also handles a validation failure two levels
// deep behind the any field (any -> *struct -> struct field), exercising
// validationSubfield's recursive case rather than just its base case.
func TestValidateAnyFieldDeeplyNested(t *testing.T) {

	type innermost struct {
		Value string `json:"value" validate:"required,max=3" vmessage:"Too long value"`
	}

	type middle struct {
		Inner innermost `json:"inner"`
	}

	type outer struct {
		Data any `json:"data"`
	}

	v := New()
	err := v.Validate(&outer{Data: &middle{Inner: innermost{Value: "way too long"}}})
	if err == nil {
		t.Fatal("expected a validation error for the deeply nested oversize field")
	}
	vErr, ok := err.(*validator.ValidationError)
	if !ok {
		t.Fatalf("expected *validator.ValidationError, got %T: %v", err, err)
	}
	if vErr.Field != "value" {
		t.Errorf("expected resolved field name %q, got %q", "value", vErr.Field)
	}
}
