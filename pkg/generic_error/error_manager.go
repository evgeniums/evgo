package generic_error

import (
	"net/http"

	"github.com/evgeniums/evgo/pkg/utils"
)

type TranslationHandler = func(string) string

type ErrorManager interface {
	ErrorDescription(code string, tr ...TranslationHandler) string
	ErrorProtocolCode(code string) int
	ErrorFamily(code string) string
	ErrorDisposition(code string) Disposition

	MakeGenericError(code string, tr ...TranslationHandler) Error

	AddErrorDescriptions(m map[string]string)
	AddErrorProtocolCodes(m map[string]int)
	AddErrorFamilies(m map[string]string)
	AddErrorDispositions(m map[string]Disposition)

	SetDefaultErrorProtocolCode(code int)
	DefaultErrorProtocolCode() int
}

type ErrorDefinitions interface {
	AttachToErrorManager(manager ErrorManager)
}

type ErrorManagerBase struct {
	descriptions        map[string]string
	protocolCodes       map[string]int
	families            map[string]string
	dispositions        map[string]Disposition
	defaultProtocolCode int
}

func (e *ErrorManagerBase) Init(defaultProtocolCode int) {
	e.defaultProtocolCode = defaultProtocolCode
	e.descriptions = make(map[string]string)
	e.protocolCodes = make(map[string]int)
	e.families = make(map[string]string)
	e.dispositions = make(map[string]Disposition)
	e.AddErrorDescriptions(CommonErrorDescriptions)
}

func (e *ErrorManagerBase) DefaultErrorProtocolCode() int {
	return e.defaultProtocolCode
}

func (e *ErrorManagerBase) SetDefaultErrorProtocolCode(code int) {
	e.defaultProtocolCode = code
}

func (e *ErrorManagerBase) AddErrorDescriptions(m map[string]string) {
	utils.AppendMap(e.descriptions, m)
}

func (e *ErrorManagerBase) AddErrorProtocolCodes(m map[string]int) {
	utils.AppendMap(e.protocolCodes, m)
}

func (e *ErrorManagerBase) AddErrorFamilies(m map[string]string) {
	utils.AppendMap(e.families, m)
}

func (e *ErrorManagerBase) AddErrorDispositions(m map[string]Disposition) {
	utils.AppendMap(e.dispositions, m)
}

func (e *ErrorManagerBase) ErrorDescription(code string, tr ...TranslationHandler) string {
	description, ok := e.descriptions[code]
	if !ok {
		description = code
	}
	if len(tr) > 0 {
		description = tr[0](description)
	}
	return description
}

func (e *ErrorManagerBase) ErrorProtocolCode(code string) int {
	protocolCode, ok := e.protocolCodes[code]
	if !ok {
		return e.defaultProtocolCode
	}
	return protocolCode
}

func (e *ErrorManagerBase) ErrorFamily(code string) string {
	return e.families[code]
}

// ErrorDisposition reports what a client should do about code, per
// whitemdesktop/docs/error-contract.md: an explicit override wins if one was registered via
// AddErrorDispositions; otherwise it is derived from the code's REGISTERED HTTP status (not the
// manager's silent default status - a code nobody classified must never be confidently reported
// as terminal just because the fallback happens to be a 4xx). A code with neither an override nor
// a registered status is DispositionUnknown.
func (e *ErrorManagerBase) ErrorDisposition(code string) Disposition {
	if d, ok := e.dispositions[code]; ok {
		return d
	}
	protocolCode, ok := e.protocolCodes[code]
	if !ok {
		return DispositionUnknown
	}
	return dispositionFromHttpStatus(protocolCode)
}

func dispositionFromHttpStatus(status int) Disposition {
	switch status {
	case http.StatusNotImplemented:
		return DispositionUnsupported
	case http.StatusUnauthorized:
		return DispositionUserAction
	case http.StatusRequestTimeout, http.StatusTooEarly, http.StatusTooManyRequests:
		return DispositionRetryAfter
	}
	switch {
	case (status >= 400 && status < 500) || status == HttpStatusClientAborted:
		return DispositionPermanent
	case status >= 500:
		return DispositionRetry
	}
	return DispositionUnknown
}

func (e *ErrorManagerBase) MakeGenericError(code string, tr ...TranslationHandler) Error {
	err := New(code, e.ErrorDescription(code, tr...))
	err.SetFamily(e.ErrorFamily(code))
	err.SetDisposition(e.ErrorDisposition(code))
	return err
}

type ErrorManagerBaseHttp struct {
	ErrorManagerBase
}

func (e *ErrorManagerBaseHttp) Init() {
	e.ErrorManagerBase.Init(http.StatusBadRequest)
	e.AddErrorDescriptions(CommonErrorDescriptions)
	e.AddErrorProtocolCodes(CommonErrorHttpCodes)
	e.AddErrorDispositions(CommonErrorDispositions)
}

type ErrorsExtender interface {
	ErrorDefinitions
	AppendErrorExtender(extender ErrorsExtender)
	Descriptions() map[string]string
	Codes() map[string]int
	Families() map[string]string
	Dispositions() map[string]Disposition
	// SetFamily stamps family on every code currently declared by this extender (i.e. every
	// key already present in Descriptions()) - so it is a package-level call, not a per-code
	// one. Call it after Init/AddErrors have populated this extender's own codes, and before
	// AppendErrorExtender pulls in any child's codes (those keep the family their own
	// package set, if any).
	SetFamily(family string)
	AddErrorDispositions(m map[string]Disposition)
}

type ErrorsExtenderBase struct {
	errorDescriptions  map[string]string
	errorProtocolCodes map[string]int
	errorFamilies      map[string]string
	errorDispositions  map[string]Disposition
}

func (e *ErrorsExtenderBase) Init(errorDescriptions map[string]string, errorProtocolCodes ...map[string]int) {
	e.errorDescriptions = errorDescriptions
	if len(errorProtocolCodes) > 0 {
		e.errorProtocolCodes = errorProtocolCodes[0]
	}
}

func (e *ErrorsExtenderBase) AddErrors(errorDescriptions map[string]string, errorProtocolCodes ...map[string]int) {

	if e.errorDescriptions == nil {
		e.Init(errorDescriptions, errorProtocolCodes...)
		return
	}

	utils.AppendMap(e.errorDescriptions, errorDescriptions)
	if len(errorProtocolCodes) > 0 {
		utils.AppendMap(e.errorProtocolCodes, errorProtocolCodes[0])
	}
}

func (e *ErrorsExtenderBase) AttachToErrorManager(manager ErrorManager) {
	manager.AddErrorDescriptions(e.errorDescriptions)
	manager.AddErrorProtocolCodes(e.errorProtocolCodes)
	manager.AddErrorFamilies(e.errorFamilies)
	manager.AddErrorDispositions(e.errorDispositions)
}

func (e *ErrorsExtenderBase) Descriptions() map[string]string {
	return e.errorDescriptions
}

func (e *ErrorsExtenderBase) Codes() map[string]int {
	return e.errorProtocolCodes
}

func (e *ErrorsExtenderBase) Families() map[string]string {
	return e.errorFamilies
}

func (e *ErrorsExtenderBase) Dispositions() map[string]Disposition {
	return e.errorDispositions
}

// SetFamily stamps family on every code this extender currently declares in Descriptions().
// See the ErrorsExtender interface doc for the required call order.
func (e *ErrorsExtenderBase) SetFamily(family string) {
	if e.errorFamilies == nil {
		e.errorFamilies = make(map[string]string, len(e.errorDescriptions))
	}
	for code := range e.errorDescriptions {
		e.errorFamilies[code] = family
	}
}

func (e *ErrorsExtenderBase) AddErrorDispositions(m map[string]Disposition) {
	if e.errorDispositions == nil {
		e.errorDispositions = make(map[string]Disposition, len(m))
	}
	utils.AppendMap(e.errorDispositions, m)
}

func (e *ErrorsExtenderBase) AppendErrorExtender(extender ErrorsExtender) {
	e.AddErrors(extender.Descriptions(), extender.Codes())
	if families := extender.Families(); len(families) > 0 {
		if e.errorFamilies == nil {
			e.errorFamilies = make(map[string]string, len(families))
		}
		utils.AppendMap(e.errorFamilies, families)
	}
	if dispositions := extender.Dispositions(); len(dispositions) > 0 {
		e.AddErrorDispositions(dispositions)
	}
}

type ErrorsExtenderStub struct {
}

func (e *ErrorsExtenderStub) AddErrors(errorDescriptions map[string]string, errorProtocolCodes ...map[string]int) {
	panic("Can't add errors to error stub")
}

func (e *ErrorsExtenderStub) AttachToErrorManager(manager ErrorManager) {
}

func (e *ErrorsExtenderStub) Descriptions() map[string]string {
	return map[string]string{}
}

func (e *ErrorsExtenderStub) Codes() map[string]int {
	return map[string]int{}
}

func (e *ErrorsExtenderStub) Families() map[string]string {
	return map[string]string{}
}

func (e *ErrorsExtenderStub) Dispositions() map[string]Disposition {
	return map[string]Disposition{}
}

func (e *ErrorsExtenderStub) SetFamily(family string) {
	panic("Can't add errors to error stub")
}

func (e *ErrorsExtenderStub) AddErrorDispositions(m map[string]Disposition) {
	panic("Can't add errors to error stub")
}

func (e *ErrorsExtenderStub) AppendErrorExtender(extender ErrorsExtender) {
	e.AddErrors(extender.Descriptions(), extender.Codes())
}
