package generic_error

import (
	"net/http"

	"github.com/evgeniums/evgo/pkg/utils"
)

type TranslationHandler = func(string) string

type ErrorManager interface {
	ErrorDescription(code string, tr ...TranslationHandler) string
	ErrorProtocolCode(code string) int

	MakeGenericError(code string, tr ...TranslationHandler) Error

	AddErrorDescriptions(m map[string]string)
	AddErrorProtocolCodes(m map[string]int)

	SetDefaultErrorProtocolCode(code int)
	DefaultErrorProtocolCode() int
}

type ErrorDefinitions interface {
	AttachToErrorManager(manager ErrorManager)
}

type ErrorManagerBase struct {
	descriptions        map[string]string
	protocolCodes       map[string]int
	defaultProtocolCode int
}

func (e *ErrorManagerBase) Init(defaultProtocolCode int) {
	e.defaultProtocolCode = defaultProtocolCode
	e.descriptions = make(map[string]string)
	e.protocolCodes = make(map[string]int)
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

func (e *ErrorManagerBase) MakeGenericError(code string, tr ...TranslationHandler) Error {
	err := New(code, e.ErrorDescription(code, tr...))
	return err
}

type ErrorManagerBaseHttp struct {
	ErrorManagerBase
}

func (e *ErrorManagerBaseHttp) Init() {
	e.ErrorManagerBase.Init(http.StatusBadRequest)
	e.AddErrorDescriptions(CommonErrorDescriptions)
	e.AddErrorProtocolCodes(CommonErrorHttpCodes)
}

type ErrorsExtender interface {
	ErrorDefinitions
	AppendErrorExtender(extender ErrorsExtender)
	Descriptions() map[string]string
	Codes() map[string]int
}

type ErrorsExtenderBase struct {
	errorDescriptions  map[string]string
	errorProtocolCodes map[string]int
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
}

func (e *ErrorsExtenderBase) Descriptions() map[string]string {
	return e.errorDescriptions
}

func (e *ErrorsExtenderBase) Codes() map[string]int {
	return e.errorProtocolCodes
}

func (e *ErrorsExtenderBase) AppendErrorExtender(extender ErrorsExtender) {
	e.AddErrors(extender.Descriptions(), extender.Codes())
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

func (e *ErrorsExtenderStub) AppendErrorExtender(extender ErrorsExtender) {
	e.AddErrors(extender.Descriptions(), extender.Codes())
}
