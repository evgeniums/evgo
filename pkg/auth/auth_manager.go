package auth

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type HandlerStore interface {
	Handler(name string) (AuthHandler, error)
	AddHandler(handler AuthHandler)
	HandlerNames() []string
}

type AuthManager interface {
	Handle(sctx context.Context, schema string) error
	ErrorDescriptions() map[string]string
	ErrorProtocolCodes() map[string]int

	Handlers() HandlerStore
	Schemas() HandlerStore
}

type HandlerFactory interface {
	Create(protocol string) (AuthHandler, error)
}

type HandlerStoreBase struct {
	handlers map[string]AuthHandler
}

func NewHandlerStore() *HandlerStoreBase {
	s := &HandlerStoreBase{}
	s.handlers = make(map[string]AuthHandler)
	return s
}

func (h *HandlerStoreBase) Handler(name string) (AuthHandler, error) {
	handler, ok := h.handlers[name]
	if !ok {
		return nil, errors.New("unknown authorization handler")
	}
	return handler, nil
}

func (h *HandlerStoreBase) HandlerNames() []string {
	return utils.AllMapKeys(h.handlers)
}

func (h *HandlerStoreBase) AddHandler(handler AuthHandler) {
	if _, ok := h.handlers[handler.Name()]; !ok {
		h.handlers[handler.Name()] = handler
	}
}

type AuthManagerBase struct {
	handlers HandlerStore
	schemas  HandlerStore
}

func (a *AuthManagerBase) Init(cfg config.Config, log logger.Logger, vld validator.Validator, handlerFactory HandlerFactory, configPath ...string) error {

	path := utils.OptionalArg("auth_manager", configPath...)
	fields := logger.Fields{"config_path": path}
	log.Debug("Init authorization manager", fields)

	a.handlers = NewHandlerStore()
	a.schemas = NewHandlerStore()

	// create and init auth methods
	log.Debug("Init auth methods")
	methodsPath := object_config.Key(path, "methods")
	methodsSection := cfg.Get(methodsPath)
	methods, ok := methodsSection.(map[string]interface{})
	if !ok {
		return log.PushFatalStack("failed to initialize authorization methods", errors.New("invalid methods section"), fields)
	}
	for methodProtocol := range methods {
		methodPath := object_config.Key(methodsPath, methodProtocol)
		fields := utils.AppendMapNew(fields, logger.Fields{"auth_method": methodProtocol, "config_path": methodPath})
		log.Debug("Init auth method", fields)
		handler, err := handlerFactory.Create(methodProtocol)
		if err != nil {
			return log.PushFatalStack("failed to create authorization method", err, fields)
		}
		err = handler.Init(cfg, log, vld, methodPath)
		if err != nil {
			return log.PushFatalStack("failed to initialize authorization method", err, fields)
		}
		handler.SetAuthManager(a)
		a.handlers.AddHandler(handler)
		for _, subHandler := range handler.Handlers() {
			a.handlers.AddHandler(subHandler)
		}
	}

	// create and init auth schemas
	log.Debug("Init auth schemas")
	schemasPath := object_config.Key(path, "schemas")
	if cfg.IsSet(schemasPath) {
		schemasSection := cfg.Get(schemasPath)
		schemas, ok := schemasSection.([]interface{})
		if !ok {
			return log.PushFatalStack("failed to initialize authorization schemas", errors.New("invalid schemas section"), fields)
		}
		for i := range schemas {
			schemaPath := object_config.KeyInt(schemasPath, i)
			fields := utils.AppendMapNew(fields, logger.Fields{"schema_path": schemaPath})
			log.Debug("Init auth schema", fields)
			schema := NewAuthSchema()
			err := schema.InitSchema(log, cfg, vld, a.handlers, schemaPath)
			if err != nil {
				return log.PushFatalStack("failed to initialize authorization schema", err, fields)
			}
			a.handlers.AddHandler(schema)
			a.schemas.AddHandler(schema)
		}
	}

	// done
	return nil
}

func (a *AuthManagerBase) ErrorDescriptions() map[string]string {
	m := utils.CopyMapOneLevel(ErrorDescriptions)
	for _, name := range a.handlers.HandlerNames() {
		handler, _ := a.handlers.Handler(name)
		utils.AppendMap(m, handler.ErrorDescriptions())
	}
	return m
}

func (a *AuthManagerBase) ErrorProtocolCodes() map[string]int {
	m := utils.CopyMapOneLevel(ErrorHttpCodes)
	for _, name := range a.handlers.HandlerNames() {
		handler, _ := a.handlers.Handler(name)
		utils.AppendMap(m, handler.ErrorProtocolCodes())
	}
	return m
}

func (a *AuthManagerBase) Handle(sctx context.Context, schema string) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AuthManagerBase.Handle", logger.Fields{"schema": schema})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// find handler
	handler, err := a.schemas.Handler(schema)
	if err != nil {
		ctx.SetGenericErrorCode(ErrorCodeInvalidAuthSchema)
		return c.SetError(err)
	}

	// run handler
	_, err = handler.Handle(sctx)
	return err
}

func (a *AuthManagerBase) Handlers() HandlerStore {
	return a.handlers
}
func (a *AuthManagerBase) Schemas() HandlerStore {
	return a.schemas
}
