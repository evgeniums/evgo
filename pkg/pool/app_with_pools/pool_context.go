package app_with_pools

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/pool"
)

type Context interface {
	op_context.Context
	Pool() pool.Pool
	SetPool(pool pool.Pool)
}

type ContextBase struct {
	op_context.Context
	pool pool.Pool
}

func (c *ContextBase) Pool() pool.Pool {
	return c.pool
}

func (c *ContextBase) SetPool(pool pool.Pool) {
	c.pool = pool
}

func (c *ContextBase) Construct(baseCtx ...op_context.Context) {
	if len(baseCtx) == 0 {
		c.Context = default_op_context.NewContext()
	} else {
		c.Context = baseCtx[0]
	}
}

func NewOpContext(fromCtx ...op_context.Context) *ContextBase {
	c := &ContextBase{}
	c.Construct(fromCtx...)
	return c
}

func NewInitOpContext(app app_context.Context, log logger.Logger, db db.DB) *ContextBase {
	c := default_op_context.NewContext()
	c.Init(app, log, db)
	t := NewOpContext(c)
	return t
}

func ContextSelfPool(ctx Context) string {
	app, ok := ctx.App().(AppWithPools)
	if !ok {
		return ""
	}
	return pool.SelfPoolName(app.Pools())
}
