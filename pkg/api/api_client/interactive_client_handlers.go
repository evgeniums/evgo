package api_client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/op_context"
	"golang.org/x/term"
)

type InteractiveClientHandlersConfig struct {
	TOKEN_FILE_NAME string `validate:"required,file"`
}

type InteractiveClientHandlers struct {
	app_context.WithAppBase
	InteractiveClientHandlersConfig
}

func (e *InteractiveClientHandlers) Config() interface{} {
	return &e.InteractiveClientHandlersConfig
}

func NewInteractiveClientHandlers(app app_context.Context) *InteractiveClientHandlers {
	e := &InteractiveClientHandlers{}
	e.WithAppBase.Init(app)
	return e
}

func (e *InteractiveClientHandlers) Init(configPath ...string) error {

	err := object_config.LoadLogValidate(e.App().Cfg(), e.App().Logger(), e.App().Validator(), e, "interactive_client", configPath...)
	if err != nil {
		return e.App().Logger().PushFatalStack("failed to load configuration of interactive api client", err)
	}

	return nil
}

func (e *InteractiveClientHandlers) GetRefreshToken() string {

	content, err := os.ReadFile(e.TOKEN_FILE_NAME)
	if err != nil || content == nil {
		e.App().Logger().Warn("failed to read refresh token from file", logger.FieldsWithError(err))
		return ""
	}

	tokenKeeper := &TokenKeeper{}
	err = json.Unmarshal(content, tokenKeeper)
	if err != nil {
		e.App().Logger().Warn("failed to unmarshal refresh token from file", logger.FieldsWithError(err))
		return ""
	}

	return tokenKeeper.Token
}

func (e *InteractiveClientHandlers) SaveRefreshToken(ctx op_context.Context, token string) {

	c := ctx.TraceInMethod("InteractiveClientHandlers.SaveRefreshToken")
	defer ctx.TraceOutMethod()

	tokenKeeper := &TokenKeeper{Token: token}

	content, err := json.MarshalIndent(tokenKeeper, "", " ")
	if err != nil {
		c.Logger().Error("failed to marshal refresh token", err)
	}

	err = os.WriteFile(e.TOKEN_FILE_NAME, content, 0644)
	if err != nil {
		c.Logger().Error("failed to save refresh token to file", err)
	}
}

func (e *InteractiveClientHandlers) GetCredentials(ctx op_context.Context) (string, string, error) {

	c := ctx.TraceInMethod("InteractiveClientHandlers.GetCredentials")
	defer ctx.TraceOutMethod()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Please, enter login:")
	login, err := reader.ReadString('\n')
	if err != nil {
		c.SetMessage("failed to enter login")
		return "", "", c.SetError(err)
	}
	login = strings.TrimSuffix(login, "\n")

	fmt.Println("Please, enter password:")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		c.SetMessage("failed to enter password")
		return "", "", c.SetError(err)
	}

	return login, string(password), nil
}
