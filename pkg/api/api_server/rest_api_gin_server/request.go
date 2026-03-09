package rest_api_gin_server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
	"github.com/gin-gonic/gin"
)

type Request struct {
	api_server.RequestBase

	server   *Server
	ginCtx   *gin.Context
	response *Response

	initialPath string
	start       time.Time

	clientIp          string
	forwardedOpSource string
}

func (r *Request) Init(s *Server, ginCtx *gin.Context, ep api_server.Endpoint, fields ...logger.Fields) {

	r.start = time.Now()
	r.server = s

	r.RequestBase.Init(s.App(), s.App().Logger(), s.App().Db(), fields...)
	r.SetEndpoint(ep)
	r.RequestBase.SetErrorManager(s)

	r.clientIp = ginCtx.ClientIP()
	if s.propagateContextId {
		ctxId := ginCtx.GetHeader(api.ForwardContext)
		if ctxId != "" {
			r.SetID(ctxId)
			r.SetLoggerField("context", ctxId)
		}
		forwardedOpSource := ginCtx.GetHeader(api.ForwardOpSource)
		if forwardedOpSource != "" {
			r.forwardedOpSource = forwardedOpSource
			r.SetLoggerField("forwarded_op_source", forwardedOpSource)
		}
	}

	r.ginCtx = ginCtx
	r.response = &Response{httpCode: http.StatusOK}
	r.response.SetRequest(r)

	r.initialPath = ginCtx.Request.URL.Path
}

func (r *Request) Server() api_server.Server {
	return r.server
}

func (r *Request) GetParameter(key string) (any, bool) {
	return r.ginCtx.Get(key)
}

func (r *Request) SetParameter(key string, value any) {
	r.ginCtx.Set(key, value)
}

func (r *Request) Response() api_server.Response {
	return r.response
}

func (r *Request) GetRequestMethod() string {
	return r.ginCtx.Request.Method
}

func (r *Request) GetRequestClientIp() string {
	return r.clientIp
}

func (r *Request) GetRequestUserAgent() string {
	return r.ginCtx.Request.UserAgent()
}

func (r *Request) Close(sctx context.Context, successMessage ...string) {
	var reponseBody interface{}
	redirect := false
	if r.response.RedirectPath() != "" {
		r.ginCtx.Redirect(http.StatusFound, r.response.RedirectPath())
		redirect = true
	}
	if r.GenericError() == nil {
		if !redirect {
			if r.response.Payload() != nil {
				r.ginCtx.String(r.response.httpCode, string(r.response.Payload()))
			} else if r.response.Message() != nil {
				reponseBody = r.response.Message()
				r.ginCtx.JSON(r.response.httpCode, r.response.Message())
			} else if r.response.File() != nil {
				file := r.response.File()
				r.ginCtx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Name))
				r.ginCtx.Header("Accept-Length", utils.NumToStr(len(file.Content)))
				r.ginCtx.Data(r.response.httpCode, file.ContentType, file.Content)
			} else if r.server.DEFAULT_RESPONSE_JSON != "" {
				r.ginCtx.String(r.response.httpCode, r.server.DEFAULT_RESPONSE_JSON)
			} else {
				r.ginCtx.Status(r.response.httpCode)
			}
		}

		r.SetLoggerField("status", "success")
	} else {
		code, err := r.server.MakeResponseError(r.GenericError())
		if code < http.StatusInternalServerError {
			r.SetErrorAsWarn(true)
		}
		reponseBody = err
		r.SetLoggerField("status", err.Code())

		if !redirect {
			r.ginCtx.JSON(code, reponseBody)
		}
	}

	if r.server.VERBOSE {
		h := r.ginCtx.Writer.Header()
		body := ""
		if reponseBody != nil {
			b, e := json.Marshal(reponseBody)
			if e == nil {
				body = string(b)
			}
		}
		r.Logger().Debug("Dump server HTTP response", logger.Fields{"response_header": h, "response_body": body})
	}

	r.RequestBase.Close(sctx, "")
	r.server.logGinRequest(r.Logger(), r.initialPath, r.start, r.ginCtx, r.LoggerFields())
}

func (r *Request) GetRequestContent() []byte {
	if r.ginCtx.Request.Body != nil {
		b, _ := io.ReadAll(r.ginCtx.Request.Body)
		r.ginCtx.Request.Body = io.NopCloser(bytes.NewBuffer(b))
		return b
	}
	return nil
}

func (r *Request) SetAuthParameter(authMethodProtocol string, key string, value string, directKeyName ...bool) {
	r.AppendResponseHeader(api_server.AuthKey(authMethodProtocol, key, directKeyName...), value)
}

func (r *Request) GetAuthParameter(authMethodProtocol string, key string, directKeyName ...bool) string {
	return r.GetRequestHeader(api_server.AuthKey(authMethodProtocol, key, directKeyName...))
}

func (r *Request) CheckRequestContent(sctx context.Context, smsMessage *string, skipSms *bool) error {
	return r.Endpoint().PrecheckBeforeAuth(sctx, smsMessage, skipSms)
}

func (r *Request) ResourceIds() api.ResourceIds {

	m := api.NewResourceIdsBase()
	for _, param := range r.ginCtx.Params {
		m.SetId(api.NewResourceIdBase(param.Key, param.Value))
	}
	return m
}

func (r *Request) GetRequestPath() string {
	return api_server.FullRequestServicePath(r)
}

func (r *Request) GetResourceId(resourceType string) api.ResourceId {
	return api.NewResourceIdBase(resourceType, r.ginCtx.Param(resourceType))
}

func (r *Request) GetTenancyId() string {
	return r.GetResourceId(tenancy_api.TenancyResource).Value()
}

func (r *Request) Validate(cmd interface{}) error {

	c := r.TraceInMethod("Request.Validate")
	defer r.TraceOutMethod()

	err := r.App().Validator().Validate(cmd)
	if err != nil {
		vErr, ok := err.(*validator.ValidationError)
		if ok {
			r.SetGenericError(vErr.GenericError(), true)
		}
		return c.SetError(err)
	}
	return nil
}

func (r *Request) ParseValidateQuery(sctx context.Context, cmd interface{}) error {

	if cmd == nil {
		return nil
	}

	c := r.TraceInMethod("Request.ParseValidateQuery")
	defer r.TraceOutMethod()

	err := http_request.ParseQuery(sctx, r.ginCtx.Request, cmd)
	if err != nil {
		return c.SetError(err)
	}

	err = r.Validate(cmd)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (r *Request) ParseValidateBody(sctx context.Context, cmd interface{}) error {

	if cmd == nil {
		return nil
	}

	c := r.TraceInMethod("Request.ParseValidateBody")
	defer r.TraceOutMethod()

	err := http_request.ParseBody(sctx, r.ginCtx.Request, cmd)
	if err != nil {
		return c.SetError(err)
	}

	err = r.Validate(cmd)
	if err != nil {
		return c.SetError(err)
	}

	return nil
}

func (r *Request) ParseAndValidate(sctx context.Context, cmd interface{}) error {

	if access_control.HttpContentInQuery(r.Endpoint().AccessType()) {
		return r.ParseValidateQuery(sctx, cmd)
	}

	return r.ParseValidateBody(sctx, cmd)
}

func (r *Request) GetGinCtx() *gin.Context {
	return r.ginCtx
}

func (r *Request) FormData() map[string][]string {
	err := r.ginCtx.Request.ParseForm()
	if err != nil {
		r.Logger().Error("failed to parse form", err)
		return map[string][]string{}
	}
	return r.ginCtx.Request.Form
}

func (r *Request) FormFile() (*multipart.FileHeader, error) {
	file, err := r.ginCtx.FormFile(r.server.FORM_SINGLE_FILE_FIELD)
	if err != nil {
		r.Logger().Error("failed to extract single file from form", err)
		return nil, err
	}
	return file, nil
}

func (r *Request) MessageFromRequest(builder func() interface{}) (interface{}, error) {
	return builder(), nil
}

func (r *Request) InjectRequestHeaders(sctx context.Context, headers map[string]string, append ...bool) context.Context {

	for k, v := range headers {
		if utils.OptionalArg(false, append...) {
			r.ginCtx.Request.Header.Add(k, v)
		} else {
			r.ginCtx.Request.Header.Set(k, v)
		}
	}

	return sctx
}

func (r *Request) GetRequestHeaders(name string) []string {
	return r.ginCtx.Request.Header.Values(name)
}

func (r *Request) GetRequestHeader(name string) string {
	// fmt.Printf("GetRequestHeader: \n")
	// for name, values := range r.ginCtx.Request.Header {
	// 	for _, value := range values {
	// 		fmt.Printf("%s: %s\n", name, value)
	// 	}
	// }
	return r.ginCtx.GetHeader(name)
}

func (r *Request) AppendResponseHeader(name string, value string) {
	r.ginCtx.Writer.Header().Add(name, value)
}

func (r *Request) GetResponseHeaders(name string) []string {
	return r.ginCtx.Writer.Header().Values(name)
}

func (r *Request) GetResponseHeader(name string) string {
	return r.ginCtx.Writer.Header().Get(name)
}
