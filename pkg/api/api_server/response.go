package api_server

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/utils"
)

type File struct {
	Content     []byte
	ContentType string
	Name        string
}

// Interface of response of server API.
type Response interface {
	Message() interface{}
	SetMessage(message api.Response)
	SetStatusMessage(status string)
	SetSuccessStatusMessage()
	Request() Request
	SetRequest(request Request)

	Payload() []byte
	SetPayload(payload []byte)

	SetRedirectPath(path string)
	RedirectPath() string

	SetFile(file *File)
	File() *File
}

type ResponseBase struct {
	message              interface{}
	request              Request
	payload              []byte
	redirectResourcePath string
	file                 *File
}

func (r *ResponseBase) Message() interface{} {
	return r.message
}

func (r *ResponseBase) SetMessage(message api.Response) {
	r.message = message
}

func (r *ResponseBase) SetStatusMessage(status string) {
	m := &api.ResponseStatus{Status: status}
	r.SetMessage(m)
}

func (r *ResponseBase) SetSuccessStatusMessage() {
	r.SetStatusMessage("success")
}

func SetResponseList(r Request, response api.ResponseListI, resourceType ...string) {

	if r.Server().IsHateoas() {
		resource := r.Endpoint().Resource()
		rType := utils.OptionalArg(resource.Type(), resourceType...)
		api.HateoasList(response, resource, rType)
	}

	r.Response().SetMessage(response)
}

func (r *ResponseBase) SetRequest(request Request) {
	r.request = request
}

func (r *ResponseBase) Request() Request {
	return r.request
}

func (r *ResponseBase) SetPayload(data []byte) {
	r.payload = data
}

func (r *ResponseBase) Payload() []byte {
	return r.payload
}

func (r *ResponseBase) SetRedirectPath(path string) {
	r.redirectResourcePath = path
}

func (r *ResponseBase) RedirectPath() string {
	return r.redirectResourcePath
}

func (r *ResponseBase) SetFile(file *File) {
	r.file = file
}

func (r *ResponseBase) File() *File {
	return r.file
}
