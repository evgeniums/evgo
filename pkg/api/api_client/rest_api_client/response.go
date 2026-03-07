package rest_api_client

import (
	"io"
	"net/http"

	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/generic_error"
)

type Response interface {
	api_client.OperationResponse
	Code() int
	Header() http.Header
	Body() []byte
	Message() string

	Error() generic_error.Error
	SetError(err generic_error.Error)
}

type HttpResponse struct {
	Raw         *http.Response
	body        []byte
	serverError generic_error.Error
}

func NewResponse(raw *http.Response) (*HttpResponse, error) {

	resp := &HttpResponse{Raw: raw}
	if resp.Code() >= http.StatusBadRequest {
		err := fillResponseError(resp)
		if err != nil {
			return resp, err
		}
	}

	return resp, nil
}

func (r *HttpResponse) Code() int {
	return r.Raw.StatusCode
}

func (r *HttpResponse) Header() http.Header {
	return r.Raw.Header
}

func (r *HttpResponse) GetHeader(key string) string {
	return r.Raw.Header.Get(key)
}

func (r *HttpResponse) SetHeader(key string, value string) {
	r.Raw.Header.Set(key, value)
}

func (r *HttpResponse) Body() []byte {
	if r.body == nil && r.Raw.Body != nil {
		r.body, _ = io.ReadAll(r.Raw.Body)
		r.Raw.Body.Close()
	}
	return r.body
}

func (r *HttpResponse) Message() string {
	return string(r.Body())
}

func (r *HttpResponse) Error() generic_error.Error {
	return r.serverError
}

func (r *HttpResponse) SetError(err generic_error.Error) {
	r.serverError = err
}

func IsResponseOK(resp Response, err error) bool {
	if err != nil || resp == nil {
		return false
	}
	return resp.Code() < http.StatusBadRequest
}
