package generic_error

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/utils"
)

// Disposition states what a client should do about an error - the terminal/retryable
// distinction that used to be inferred client-side from message text or HTTP status. See
// whitemdesktop/docs/error-contract.md for the full contract this implements.
type Disposition string

const (
	// DispositionUnknown means the server did not state a disposition - either because the
	// code has no registered HTTP status to derive one from, or because the peer predates
	// this contract. It is the zero value: a client must fall back to its own heuristics,
	// exactly as if this field did not exist.
	DispositionUnknown Disposition = ""
	// DispositionPermanent means this request will never succeed as issued.
	DispositionPermanent Disposition = "permanent"
	// DispositionUnsupported means the server does not implement this call, or the API
	// version is too old. Terminal like DispositionPermanent, but distinct: a client should
	// stop offering the feature, not just fail this one call.
	DispositionUnsupported Disposition = "unsupported"
	// DispositionRetry means the failure is transient; retry with backoff.
	DispositionRetry Disposition = "retry"
	// DispositionRetryAfter means retryable, but not yet - RetryAfter carries the delay in
	// seconds.
	DispositionRetryAfter Disposition = "retry_after"
	// DispositionUserAction means retryable only after the user does something: re-auth,
	// free storage, raise a quota.
	DispositionUserAction Disposition = "user_action"
)

// IsTerminal reports whether a client must not retry automatically.
func (d Disposition) IsTerminal() bool {
	return d == DispositionPermanent || d == DispositionUnsupported
}

// IsStated reports whether the server expressed an opinion at all.
func (d Disposition) IsStated() bool {
	return d != DispositionUnknown
}

// Generic error that can be forwarded from place of arising to place of user reporting.
type Error interface {
	error
	Code() string
	Message() string
	Details() string
	Family() string
	Disposition() Disposition
	RetryAfter() int
	Original() error
	Data() interface{}

	SetMessage(msg string)
	SetDetails(details string)
	SetFamily(value string)
	SetDisposition(value Disposition)
	SetRetryAfter(seconds int)
	SetOriginal(err error)

	SetData(data interface{})
}

type ErrorHolder struct {
	Code        string      `json:"code" validate:"omitempty,alphanum_,max=64" vmessage:"Invalid error code"`
	Message     string      `json:"message"`
	Details     string      `json:"details,omitempty"`
	Family      string      `json:"family,omitempty"`
	Disposition Disposition `json:"disposition,omitempty"`
	RetryAfter  int         `json:"retry_after,omitempty"`
	Original    error       `json:"-"`
	Data        interface{} `json:"data,omitempty"`
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

func (e *ErrorBase) SetDisposition(value Disposition) {
	e.ErrorHolder.Disposition = value
}

func (e *ErrorBase) Disposition() Disposition {
	return e.ErrorHolder.Disposition
}

func (e *ErrorBase) SetRetryAfter(seconds int) {
	e.ErrorHolder.RetryAfter = seconds
}

func (e *ErrorBase) RetryAfter() int {
	return e.ErrorHolder.RetryAfter
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
		return e
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
