package generic_error

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/utils"
)

// Generic error that can be forwarded from place of arising to place of user reporting.
type Error interface {
	error
	Code() string
	Message() string
	Details() string
	Family() string
	Original() error
	Data() interface{}

	SetMessage(msg string)
	SetDetails(details string)
	SetFamily(value string)
	SetOriginal(err error)

	SetData(data interface{})
}

type ErrorHolder struct {
	Code     string      `json:"code" validate:"omitempty,alphanum_,max=64" vmessage:"Invalid error code"`
	Message  string      `json:"message"`
	Details  string      `json:"details,omitempty"`
	Family   string      `json:"family,omitempty"`
	Original error       `json:"-"`
	Data     interface{} `json:"data,omitempty"`
}

type ErrorBase struct {
	ErrorHolder
}

func NewEmpty() *ErrorBase {
	return &ErrorBase{}
}

// Create new error from code and message.
func New(code string, message ...string) *ErrorBase {
	e := &ErrorBase{ErrorHolder{Code: code}}
	if len(message) > 0 {
		e.ErrorHolder.Message = message[0]
	}
	return e
}

// Create new error from code and message taken from other "native error".
func NewFromErr(err error, code ...string) *ErrorBase {
	return New(utils.OptionalArg(ErrorCodeUnknown, code...), err.Error())
}

// Create new error from code, message and some other "original error" with keeping native error.
func NewFromOriginal(code string, message string, original error) *ErrorBase {
	e := &ErrorBase{ErrorHolder{Code: code, Message: message, Original: original}}
	return e
}

// Create new error from message.
func NewFromMessage(message string) *ErrorBase {
	e := &ErrorBase{ErrorHolder{Code: ErrorCodeUnknown, Message: message}}
	return e
}

// Convert error to string for error interface.
func (e *ErrorBase) Error() string {
	if e.ErrorHolder.Original != nil {
		return fmt.Sprintf("%s: %s", e.ErrorHolder.Message, e.ErrorHolder.Original)
	}
	return e.ErrorHolder.Message
}

// Get error code.
func (e *ErrorBase) Code() string {
	return e.ErrorHolder.Code
}

// Convert error message.
func (e *ErrorBase) Message() string {
	return e.ErrorHolder.Message
}

// Set error message.
func (e *ErrorBase) SetMessage(message string) {
	e.ErrorHolder.Message = message
}

// Get error details.
func (e *ErrorBase) Details() string {
	return e.ErrorHolder.Details
}

func (e *ErrorBase) SetFamily(value string) {
	e.ErrorHolder.Family = value
}

func (e *ErrorBase) Family() string {
	return e.ErrorHolder.Family
}

// Set error details.
func (e *ErrorBase) SetDetails(details string) {
	e.ErrorHolder.Details = details
}

// Get original error.
func (e *ErrorBase) Original() error {
	return e.ErrorHolder.Original
}

// Set original error.
func (e *ErrorBase) SetOriginal(err error) {
	e.ErrorHolder.Original = err
}

// Extract code from the error. If error is not of Error type then code is unknown_error.
func Code(e error) string {
	if e == nil {
		return ""
	}
	err, ok := e.(Error)
	if !ok {
		return ErrorCodeUnknown
	}
	return err.Code()
}

// Extract message from the error. If error is not of Error type then error as string is used.
func Message(e error) string {
	if e == nil {
		return ""
	}
	err, ok := e.(Error)
	if !ok {
		return e.Error()
	}
	return err.Error()
}

// Extract details from the error.
func Details(e error) string {
	if e == nil {
		return ""
	}
	err, ok := e.(Error)
	if !ok {
		return ""
	}
	return err.Details()
}

// Extract original error from the error. If error is not of Error type then the argument is returned as is.
func Original(e error) error {
	if e == nil {
		return nil
	}
	err, ok := e.(Error)
	if !ok {
		return err
	}
	return err.Original()
}

// Set error data.
func (e *ErrorBase) SetData(data interface{}) {
	e.ErrorHolder.Data = data
}

// Get error data.
func (e *ErrorBase) Data() interface{} {
	return e.ErrorHolder.Data
}

func MapErrorData(e Error, obj interface{}) error {
	respMap, ok := e.Data().(map[string]interface{})
	if ok {
		err := utils.MapToStruct(respMap, obj)
		if err != nil {
			return err
		}
	}
	return nil
}
