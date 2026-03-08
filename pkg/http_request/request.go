package http_request

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/message/message_json"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/gorilla/schema"
)

const MaxDumpSize int = 2048

type Request struct {
	NativeRequest         *http.Request
	NativeResponse        *http.Response
	ResponseStatus        int
	Body                  []byte
	ResponseBody          []byte
	ResponseContent       string
	GoodResponse          interface{}
	BadResponse           interface{}
	Transport             http.RoundTripper
	Timeout               int
	ParsingFailed         bool
	IgnoreResponseContent bool

	client *http.Client

	TxSerializer message.Serializer
	RxSerializer message.Serializer
	UserAgent    string
}

func (r *Request) SetSerializer(serializer ...message.Serializer) {
	if len(serializer) == 1 {
		r.TxSerializer = serializer[0]
		r.RxSerializer = serializer[0]
	} else if len(serializer) == 2 {
		r.TxSerializer = serializer[0]
		r.RxSerializer = serializer[1]
	} else {
		r.TxSerializer = message_json.Serializer
		r.RxSerializer = message_json.Serializer
	}
}

func NewPostWithContext(systemCtx context.Context, sctx context.Context, url string, msg interface{}, serializer ...message.Serializer) (*Request, error) {

	r := &Request{}
	r.SetSerializer(serializer...)

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("http_request.NewPost", logger.Fields{"url": url})
	defer ctx.TraceOutMethod()

	var cmdByte []byte
	var err error

	var body io.Reader
	if msg != nil {
		cmdByte, err = r.TxSerializer.SerializeMessage(msg)
		if err != nil {
			c.SetMessage("failed to marshal message")
			return nil, c.SetError(err)
		}
		r.Body = cmdByte
		body = bytes.NewBuffer(cmdByte)
	}

	r.NativeRequest, err = http.NewRequestWithContext(systemCtx, http.MethodPost, url, body)
	if err != nil {
		c.SetMessage("failed to create request")
		return nil, c.SetError(err)
	}

	if r.UserAgent != "" {
		r.NativeRequest.Header.Set("User-Agent", r.UserAgent)
	}

	if r.TxSerializer.ContentMime() != "" {
		r.NativeRequest.Header.Set("Content-Type", utils.ConcatStrings(r.TxSerializer.ContentMime(), ";charset=UTF-8"))
	}

	return r, nil
}

func NewPost(sctx context.Context, url string, msg interface{}, serializer ...message.Serializer) (*Request, error) {
	return NewPostWithContext(context.Background(), sctx, url, msg, serializer...)
}

func UrlEncode(msg interface{}) (string, error) {
	if msg != nil {
		encoder := schema.NewEncoder()
		encoder.SetAliasTag("json")
		encoder.RegisterEncoder(utils.DateNil, utils.DateReflectStr)
		encoder.RegisterEncoder(utils.TimeNil, utils.TimeReflectStr)
		v := url.Values{}
		err := encoder.Encode(msg, v)
		if err != nil {
			return "", err
		}
		return v.Encode(), nil
	}
	return "", nil
}

func NewGetWithContext(systemCtx context.Context, sctx context.Context, uRL string, msg interface{}, serializer ...message.Serializer) (*Request, error) {

	r := &Request{}
	r.SetSerializer(serializer...)

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("http_request.NewGet", logger.Fields{"url": uRL})
	defer ctx.TraceOutMethod()

	var err error

	if strings.Contains(uRL, "?") {
		err = errors.New("URL must not contain query string, encode query in msg object")
		return nil, err
	}

	r.NativeRequest, err = http.NewRequestWithContext(systemCtx, http.MethodGet, uRL, nil)
	if err != nil {
		c.SetMessage("failed to create request")
		return nil, c.SetError(err)
	}

	query, err := UrlEncode(msg)
	if err != nil {
		c.SetMessage("failed to build query")
		return nil, c.SetError(err)
	}

	r.NativeRequest.URL.RawQuery = query

	return r, nil
}

func NewGet(sctx context.Context, uRL string, msg interface{}) (*Request, error) {
	return NewGetWithContext(sctx, sctx, uRL, msg)
}

func (r *Request) SendRaw(sctx context.Context) error {

	// TODO set user agent

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Request.SendRaw", logger.Fields{"url": r.NativeRequest.URL.String(), "method": r.NativeRequest.Method})
	defer ctx.TraceOutMethod()
	var err error

	// TODO use this flag for server
	if ctx.Logger().DumpRequests() {
		dump, _ := httputil.DumpRequestOut(r.NativeRequest, true)
		c.Logger().Debug("Client dump HTTP request", logger.Fields{"http_request": string(dump)})
	}

	client := r.client
	if client == nil {
		client = &http.Client{Transport: r.Transport}
		if r.Timeout != 0 {
			client.Timeout = time.Second * time.Duration(r.Timeout)
		}
	}
	r.NativeResponse, err = client.Do(r.NativeRequest)

	if ctx.Logger().DumpRequests() {
		if r.NativeResponse != nil {
			dump, _ := httputil.DumpResponse(r.NativeResponse, true)
			c.Logger().Debug("Client dump HTTP response", logger.Fields{"http_response": string(dump)})
		} else {
			c.Logger().Debug("Client dump HTTP response", logger.Fields{"http_response": ""})
		}
	}

	if err != nil {
		c.SetLoggerField("http_response_nil", r.NativeResponse == nil)
		return c.SetError(err)
	}

	return nil
}

func (r *Request) Send(sctx context.Context, relaxedParsing ...bool) error {

	// TODO set user agent

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("Request.Send", logger.Fields{"url": r.NativeRequest.URL.String(), "method": r.NativeRequest.Method})

	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	if ctx.Logger().DumpRequests() {
		dump, err1 := httputil.DumpRequestOut(r.NativeRequest, true)
		if err1 != nil {
			c.Logger().Error("Failed to dump HTTP request", err1)
		} else {
			c.Logger().Debug("Client dump HTTP request", logger.Fields{"http_request": string(dump)})
		}
	}

	client := r.client
	if client == nil {
		client = &http.Client{Transport: r.Transport}
		if r.Timeout != 0 {
			client.Timeout = time.Second * time.Duration(r.Timeout)
		}
	}
	r.NativeResponse, err = client.Do(r.NativeRequest)

	if ctx.Logger().DumpRequests() {
		if r.NativeResponse != nil {
			dump, err1 := httputil.DumpResponse(r.NativeResponse, true)
			if err1 != nil {
				c.Logger().Error("failed to dump HTTP response", err1)
			} else {
				// TODO make it singleton
				dumpStr := string(dump)
				text := utils.Substr(string(dump), 0, MaxDumpSize)
				if len(text) < len(dumpStr) {
					text = utils.ConcatStrings(text, "...")
				}
				c.Logger().Debug("Client dump HTTP response", logger.Fields{"http_response": text})
			}
		} else {
			c.Logger().Debug("Client dump HTTP response", logger.Fields{"http_response": ""})
		}
	}

	if err != nil {
		// TODO catch cancel and timeout
		c.SetMessage("failed to client.Do")
		return err
	}

	if r.NativeResponse != nil {
		r.ResponseStatus = r.NativeResponse.StatusCode
		if r.NativeResponse.Body != nil {
			var body []byte
			body, _ = io.ReadAll(r.NativeResponse.Body)
			r.NativeResponse.Body.Close()

			r.ResponseBody = body
			if !r.IgnoreResponseContent || r.ResponseStatus >= http.StatusBadRequest {
				r.ResponseContent = string(r.ResponseBody)
			}

			parseResponse := func(obj interface{}) {
				if r.RxSerializer == nil {
					r.RxSerializer = message_json.Serializer
				}
				err = r.RxSerializer.ParseMessage(body, obj)
				if err != nil {
					r.ParsingFailed = true
				}
			}

			if r.ResponseStatus < http.StatusBadRequest {
				if r.GoodResponse != nil {
					parseResponse(r.GoodResponse)
					if err != nil {
						if !utils.OptionalArg(false, relaxedParsing...) {
							c.SetMessage("failed to parse good response")
						} else {
							err = nil
						}
					}
				}
			} else {
				if r.BadResponse != nil {
					parseResponse(r.BadResponse)
					if err != nil {
						if !utils.OptionalArg(false, relaxedParsing...) {
							c.SetMessage("failed to parse bad response")
						} else {
							err = nil
						}
					}
				}
			}

			if err != nil {
				// set logger fields only if parsing fails
				if !ctx.Logger().DumpRequests() {
					text := utils.Substr(r.ResponseContent, 0, MaxDumpSize)
					if len(text) < len(r.ResponseContent) {
						text = utils.ConcatStrings(text, "...")
					}
					c.SetLoggerField("response_content", text)
				}
				c.SetLoggerField("response_status", r.ResponseStatus)
				return err
			}
		}
	}

	return nil
}

func (r *Request) SetHeader(key string, value string) {
	r.NativeRequest.Header.Set(key, value)
}

func (r *Request) SetAuthHeader(key string, value string) {
	str := fmt.Sprintf("%s %s", key, value)
	r.NativeRequest.Header.Set("Authorization", str)
}

func HttpHeadersSet(req *http.Request, headers ...map[string]string) {
	if len(headers) > 0 {
		for k, v := range headers[0] {
			req.Header.Set(k, v)
		}
	}
}

func NewMultipart(sctx context.Context, url string, files map[string]io.Reader, fields map[string]string, filesField ...string) (*Request, error) {

	// prepare
	filesFieldName := utils.OptionalArg("files", filesField...)
	r := &Request{}
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("http_request.NewMultipart", logger.Fields{"url": url})
	defer ctx.TraceOutMethod()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// write other fields
	for key, r := range fields {
		err := w.WriteField(key, r)
		if err != nil {
			c.SetLoggerField("field", key)
			c.SetMessage("failed to create form field")
			return nil, c.SetError(err)
		}
	}

	// write files
	for key, r := range files {
		fw, err := w.CreateFormFile(filesFieldName, key)
		if err != nil {
			c.SetLoggerField("file", key)
			c.SetMessage("failed to create form file")
			return nil, c.SetError(err)
		}
		_, err = io.Copy(fw, r)
		if err != nil {
			c.SetLoggerField("file", key)
			c.SetMessage("failed to read file")
			return nil, c.SetError(err)
		}
	}
	w.Close()

	// create request
	var err error
	r.NativeRequest, err = http.NewRequest(http.MethodPost, url, &b)
	if err != nil {
		c.SetMessage("failed to create request")
		return nil, c.SetError(err)
	}

	// set content type
	r.NativeRequest.Header.Set("Content-Type", w.FormDataContentType())

	// done
	return r, nil
}
