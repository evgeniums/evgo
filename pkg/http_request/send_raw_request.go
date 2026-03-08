package http_request

import (
	"context"
	"net/http"
	"net/http/httputil"

	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type RedirectHandler func(req *http.Request, via []*http.Request) error

func SendRawRequest(sctx context.Context, request *http.Request, redirectHandler ...RedirectHandler) (*http.Response, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("http_request.Send", logger.Fields{"url": request.URL.Path, "method": request.Method})
	defer ctx.TraceOutMethod()

	// TODO use this flag for server
	if ctx.Logger().DumpRequests() {
		dump, _ := httputil.DumpRequestOut(request, true)
		c.Logger().Debug("Dump raw HTTP request", logger.Fields{"http_request": string(dump)})
	}

	client := &http.Client{}
	if len(redirectHandler) != 0 {
		client.CheckRedirect = redirectHandler[0]
	}
	response, err := client.Do(request)

	if ctx.Logger().DumpRequests() {
		if response != nil {
			dump, _ := httputil.DumpResponse(response, true)
			c.Logger().Debug("Dump raw HTTP response", logger.Fields{"http_response": string(dump)})
		} else {
			c.Logger().Debug("Dump raw HTTP response", logger.Fields{"http_response": ""})
		}
	}

	if err != nil {
		c.SetLoggerField("http_response_nil", response == nil)
		return nil, c.SetError(err)
	}

	return response, nil
}
