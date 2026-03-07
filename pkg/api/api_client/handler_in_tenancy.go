package api_client

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/multitenancy"
)

type HandlerInTenancy[Cmd any, Result any] struct {
	Cmd    *Cmd
	Result *Result
}

func (h *HandlerInTenancy[Cmd, Result]) Exec(client Client, ctx multitenancy.TenancyContext, operation api.Operation) error {

	c := ctx.TraceInMethod("HandlerInTenancy.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, h.Cmd, h.Result, ctx.GetTenancy())
	if err != nil {
		return c.SetError(err)
	}
	return nil
}

func NewHandlerInTenancy[Cmd any, Result any](cmd *Cmd, result *Result) *HandlerInTenancy[Cmd, Result] {
	e := &HandlerInTenancy[Cmd, Result]{Cmd: cmd, Result: result}
	return e
}

type HandlerInTenancyCmd[Cmd any] struct {
	Cmd *Cmd
}

func (h *HandlerInTenancyCmd[Cmd]) Exec(client Client, ctx multitenancy.TenancyContext, operation api.Operation) error {

	c := ctx.TraceInMethod("HandlerInTenancy.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, h.Cmd, nil, ctx.GetTenancy())
	if err != nil {
		return c.SetError(err)
	}
	return nil
}

func NewHandlerInTenancyCmd[Cmd any](cmd *Cmd) *HandlerInTenancyCmd[Cmd] {
	e := &HandlerInTenancyCmd[Cmd]{Cmd: cmd}
	return e
}

type HandlerInTenancyResult[Result any] struct {
	Result *Result
}

func (h *HandlerInTenancyResult[Result]) Exec(client Client, ctx multitenancy.TenancyContext, operation api.Operation) error {

	c := ctx.TraceInMethod("HandlerInTenancy.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, nil, h.Result, ctx.GetTenancy())
	c.SetError(err)
	return err
}

func NewHandlerInTenancyResult[Result any](result *Result) *HandlerInTenancyResult[Result] {
	e := &HandlerInTenancyResult[Result]{Result: result}
	return e
}

type HandlerInTenancyNil struct {
}

func (h *HandlerInTenancyNil) Exec(client Client, ctx multitenancy.TenancyContext, operation api.Operation) error {

	c := ctx.TraceInMethod("HandlerInTenancy.Exec")
	defer ctx.TraceOutMethod()

	err := client.Exec(ctx, operation, nil, nil, ctx.GetTenancy())
	c.SetError(err)
	return err
}

func NewHandlerInTenancyNil() *HandlerInTenancyNil {
	e := &HandlerInTenancyNil{}
	return e
}
