package http_request

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/message/message_json"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/gorilla/schema"
)

func ParseQuery(ctx op_context.Context, request *http.Request, cmd interface{}) error {
	c := ctx.TraceInMethod("http_request.ParseQuery", logger.Fields{"query": request.URL.RawQuery})
	defer ctx.TraceOutMethod()

	if request.URL.RawQuery == "" {
		return nil
	}

	vals, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		c.SetMessage("failed to parse query")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		return c.SetError(err)
	}

	decoder := schema.NewDecoder()
	decoder.SetAliasTag("json")
	decoder.RegisterConverter(utils.DateNil, utils.DateConverter)
	decoder.RegisterConverter(utils.TimeNil, utils.TimeConverter)
	decoder.IgnoreUnknownKeys(true)
	err = decoder.Decode(cmd, vals)
	if err != nil {
		c.SetLoggerField("query_vals", fmt.Sprintf("%+v", vals))
		c.SetMessage("failed to decode schema")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		return c.SetError(err)
	}

	return nil
}

func ParseBody(ctx op_context.Context, request *http.Request, cmd interface{}, serializer ...message.Serializer) error {

	c := ctx.TraceInMethod("http_request.ParseBody")
	defer ctx.TraceOutMethod()

	s := utils.OptionalArg[message.Serializer](message_json.Serializer, serializer...)

	var body []byte
	if request.Body != nil {
		body, _ = io.ReadAll(request.Body)
		if body != nil {
			request.Body = io.NopCloser(bytes.NewBuffer(body))
		}
	}

	err := s.ParseMessage(body, cmd)
	if err != nil {
		c.SetMessage("failed to parse body")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeFormat)
		return c.SetError(err)
	}

	return nil
}
