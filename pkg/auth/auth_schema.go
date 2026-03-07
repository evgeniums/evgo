package auth

import (
	"errors"
	"fmt"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type Aggregation = string

const AggregationProtocol = "aggregation"

const (
	And Aggregation = "and"
	Or  Aggregation = "or"
)

type AuthSchemaConfig struct {
	NAME        string `validate:"required"`
	AGGREGATION string `default:"and" validate:"omitempty,oneof=and or"`
}

type AuthSchema struct {
	AuthHandlerBase
	config   *AuthSchemaConfig
	handlers []AuthHandler
}

func NewAuthSchema() *AuthSchema {
	s := &AuthSchema{}
	s.Construct()
	return s
}

func (s *AuthSchema) Construct() {
	s.config = &AuthSchemaConfig{}
	s.handlers = make([]AuthHandler, 0)
}

func (a *AuthSchema) Config() interface{} {
	return a.config
}

func (a *AuthSchema) Aggregation() Aggregation {
	return a.config.AGGREGATION
}

func (a *AuthSchema) SetAggregation(aggregation Aggregation) {
	a.config.AGGREGATION = aggregation
}

func (a *AuthSchema) Handlers() []AuthHandler {
	return a.handlers
}

func (a *AuthSchema) AppendHandlers(handler ...AuthHandler) {
	a.handlers = append(a.handlers, handler...)
}

func (a *AuthSchema) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {
	return errors.New("use AuthSchema.InitSchema for initialization")
}

func (a *AuthSchema) InitSchema(log logger.Logger, cfg config.Config, vld validator.Validator, handlerStore HandlerStore, configPath ...string) error {

	path := utils.OptionalArg("auth_schema", configPath...)
	fields := logger.Fields{"path": path}

	// load plain configuration
	err := object_config.LoadLogValidate(cfg, log, vld, a, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of authorization schema", err, fields)
	}
	a.SetName(a.config.NAME)

	// load handlers
	key := object_config.Key(path, "handlers")
	handlersSection := cfg.Get(key)
	handlers, ok := handlersSection.([]interface{})
	if !ok {
		err = errors.New("invalid handlers section")
		return log.PushFatalStack("failed to load configuration of authorization schema", err, fields)
	}
	for i := range handlers {
		handlerPath := object_config.Key(key, fmt.Sprintf("%d", i))
		fields := logger.Fields{"path": handlerPath}

		handlerName := cfg.GetString(object_config.Key(handlerPath, "name"))
		if handlerName != "" {
			fields["handler"] = handlerName
			handler, err := handlerStore.Handler(handlerName)
			if err != nil {
				return log.PushFatalStack("failed to find authorization handler in handlers store", err, fields)
			}
			a.handlers = append(a.handlers, handler)
		} else {
			childSchema := &AuthSchema{}
			err = childSchema.InitSchema(log, cfg, vld, handlerStore, handlerPath)
			if err != nil {
				return err
			}
			a.handlers = append(a.handlers, childSchema)
			if childSchema.Name() != "" {
				handlerStore.AddHandler(childSchema)
			}
		}
	}

	// done
	return nil
}

func (a *AuthSchema) Handle(ctx AuthContext) (bool, error) {

	// setup
	c := ctx.TraceInMethod("AuthSchema.Handle", logger.Fields{"path": ctx.GetRequestPath()})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	authMethodFound := false
	for _, handler := range a.handlers {
		c.SetLoggerField("auth_method", handler.Name())
		authMethodFound, err = handler.Handle(ctx)
		if !authMethodFound {
			if a.config.AGGREGATION == Or {
				continue
			}
			ctx.SetGenericErrorCode(ErrorCodeUnauthorized)
			if err == nil {
				err = errors.New("auth method not found in request")
			}
			return false, err
		}
		if err != nil {
			ctx.SetGenericErrorCode(ErrorCodeUnauthorized)
			return true, err
		}
		if a.config.AGGREGATION == Or {
			return true, nil
		}
	}

	return len(a.handlers) == 0, nil
}

func (a *AuthSchema) Protocol() string {
	return AggregationProtocol
}
