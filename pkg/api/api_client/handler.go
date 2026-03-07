package api_client

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type Handler[Request any, Result any] struct {
	Request *Request
	Result  *Result
}

func (h *Handler[Request, Result]) Exec(client Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("Handler.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, h.Request, h.Result)
	c.SetError(err)
	return err
}

func NewHandler[Request any, Result any](request *Request, result *Result) *Handler[Request, Result] {
	e := &Handler[Request, Result]{Request: request, Result: result}
	return e
}

type HandlerRequest[Request any] struct {
	Request *Request
}

func (h *HandlerRequest[Request]) Exec(client Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("Handler.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, h.Request, nil)
	c.SetError(err)
	return err
}

func NewHandlerRequest[Request any](request *Request) *HandlerRequest[Request] {
	e := &HandlerRequest[Request]{Request: request}
	return e
}

type HandlerResult[Result any] struct {
	Result *Result
}

func (h *HandlerResult[Result]) Exec(client Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("Handler.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, nil, h.Result)
	c.SetError(err)
	return err
}

func NewHandlerResult[Result any](result *Result) *HandlerResult[Result] {
	e := &HandlerResult[Result]{Result: result}
	return e
}

type HandlerNil struct {
}

func (h *HandlerNil) Exec(client Client, ctx op_context.Context, operation api.Operation) error {

	c := ctx.TraceInMethod("Handler.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, nil, nil)
	c.SetError(err)
	return err
}

func NewHandlerNil() *HandlerNil {
	e := &HandlerNil{}
	return e
}
