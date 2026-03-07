package api_client

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type EmbeddedClientHandlersConfig struct {
	TOKEN_CACHE_KEY string `validate:"required" default:"client_refresh_token"`
	LOGIN           string `validate:"required"`
	PASSWORD        string `mask:"true"`
}

type EmbeddedClientHandlers struct {
	app_context.WithAppBase
	EmbeddedClientHandlersConfig
}

func (e *EmbeddedClientHandlers) Config() interface{} {
	return &e.EmbeddedClientHandlersConfig
}

func NewEmbeddedClientHandlers(app app_context.Context) *EmbeddedClientHandlers {
	e := &EmbeddedClientHandlers{}
	e.WithAppBase.Init(app)
	return e
}

func (e *EmbeddedClientHandlers) InitFromConfig(configPath ...string) error {

	err := object_config.LoadLogValidate(e.App().Cfg(), e.App().Logger(), e.App().Validator(), e, "embedded_client", configPath...)
	if err != nil {
		return e.App().Logger().PushFatalStack("failed to load configuration of embedded api client", err)
	}

	return nil
}

func (e *EmbeddedClientHandlers) InitDirect(ctx op_context.Context, login string, password string, tokenCacheKey string) error {

	e.LOGIN = login
	e.PASSWORD = password
	e.TOKEN_CACHE_KEY = tokenCacheKey

	return nil
}

type TokenKeeper struct {
	Token string `json:"token"`
}

func (e *EmbeddedClientHandlers) GetRefreshToken() string {
	tokenKeeper := &TokenKeeper{}
	found, err := e.App().Cache().Get(e.TOKEN_CACHE_KEY, tokenKeeper)
	if found && err != nil {
		return tokenKeeper.Token
	}
	e.App().Logger().Warn("client refresh token not found in cache")
	return ""
}

func (e *EmbeddedClientHandlers) SaveRefreshToken(ctx op_context.Context, token string) {
	tokenKeeper := &TokenKeeper{Token: token}
	err := e.App().Cache().Set(e.TOKEN_CACHE_KEY, tokenKeeper)
	if err != nil {
		ctx.Logger().Error("failed to save client refresh token in cache", err)
	}
}

func (e *EmbeddedClientHandlers) GetCredentials(ctx op_context.Context) (login string, password string, err error) {
	return e.LOGIN, e.PASSWORD, nil
}
