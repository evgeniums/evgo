package auth

import (
	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type Auth interface {
	generic_error.ErrorDefinitions
	HandleRequest(ctx AuthContext, path string, access access_control.AccessType) error
}

type EndpointsAuth interface {
	Auth
	Manager() AuthManager
	EndpointsConfig() EndpointsAuthConfig
}

type AuthBaseConfig struct {
	DEFAULT_SCHEMA string `default:"check_token"`
}

type AuthBase struct {
	AuthBaseConfig
	manager         AuthManager
	endpointsConfig EndpointsAuthConfig
}

func NewAuth() *AuthBase {
	return &AuthBase{}
}

func (a *AuthBase) Config() interface{} {
	return &a.AuthBaseConfig
}

func (a *AuthBase) Init(cfg config.Config, log logger.Logger, vld validator.Validator, handlerFactory HandlerFactory, configPath ...string) error {

	path := utils.OptionalArg("auth", configPath...)

	err := object_config.LoadLogValidate(cfg, log, vld, a, path)
	if err != nil {
		return log.PushFatalStack("failed to load auth configuration", err)
	}

	manager := &AuthManagerBase{}
	err = manager.Init(cfg, log, vld, handlerFactory, object_config.Key(path, "manager"))
	if err != nil {
		return err
	}
	a.manager = manager

	epConfig := &EndpointsAuthConfigBase{}
	err = epConfig.Init(cfg, log, vld, object_config.Key(path, "endpoints"))
	if err != nil {
		return err
	}
	a.endpointsConfig = epConfig

	return nil
}

func (a *AuthBase) HandleRequest(ctx AuthContext, path string, access access_control.AccessType) error {

	// setup
	c := ctx.TraceInMethod("AuthBase.Handle", logger.Fields{"auth_path": path})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find schema
	schema, ok := a.endpointsConfig.Schema(path, access)
	if !ok {
		schema = a.DEFAULT_SCHEMA
	}

	// run handler
	err = a.manager.Handle(ctx, schema)
	if err != nil {
		return err
	}

	// done
	return nil
}

func (a *AuthBase) AttachToErrorManager(errManager generic_error.ErrorManager) {
	errManager.AddErrorDescriptions(a.manager.ErrorDescriptions())
	errManager.AddErrorProtocolCodes(a.manager.ErrorProtocolCodes())
}

func (a *AuthBase) EndpointsConfig() EndpointsAuthConfig {
	return a.endpointsConfig
}

func (a *AuthBase) Manager() AuthManager {
	return a.manager
}

type WithAuth interface {
	Auth() Auth
}

type WithAuthBase struct {
	auth Auth
}

func (w *WithAuthBase) Init(auth Auth) {
	w.auth = auth
}

func (w *WithAuthBase) Auth() Auth {
	return w.auth
}
