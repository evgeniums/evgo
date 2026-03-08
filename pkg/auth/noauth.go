package auth

import (
	"context"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/validator"
)

const NoAuthProtocol = "noauth"

type NoAuthMethod struct {
	AuthHandlerBase
}

func (n *NoAuthMethod) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	n.AuthHandlerBase.Init(NoAuthProtocol)

	return nil
}

func (n *NoAuthMethod) Handle(sctx context.Context) (bool, error) {

	ctx := op_context.OpContext[AuthContext](sctx)
	ctx.TraceInMethod("NoAuth.Handle")
	defer ctx.TraceOutMethod()

	user := &UserBase{}
	user.UserId = ""
	user.UserDisplay = ""
	user.UserLogin = ""
	ctx.SetAuthUser(user)

	return true, nil
}

func (n *NoAuthMethod) SetAuthManager(manager AuthManager) {
	manager.Schemas().AddHandler(n)
}

type NoAuth struct {
	handler *NoAuthMethod
}

func NewNoAuth() *NoAuth {
	a := &NoAuth{}
	a.handler = &NoAuthMethod{}
	return a
}

func (a *NoAuth) HandleRequest(sctx context.Context, path string, access access_control.AccessType) error {
	a.handler.Handle(sctx)
	return nil
}

func (a *NoAuth) AttachToErrorManager(errManager generic_error.ErrorManager) {
}
