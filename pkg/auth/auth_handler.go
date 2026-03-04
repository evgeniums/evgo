package auth

import (
	"strings"

	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
)

const AuthorizationHeader = "authorization"
const AuthorizationBearer = "bearer"

type AuthDataAccessor interface {
	Set(key string, value string)
	Get(key string) string
}

type AuthContext interface {
	UserContext
	Session

	GetRequestContent() []byte
	CheckRequestContent(smsMessage *string, skipSms *bool) error
	GetRequestPath() string
	GetRequestMethod() string
	GetResourceId(resourceType string) api.ResourceId
	ResourceIds() api.ResourceIds

	GetTenancyId() string

	GetRequestClientIp() string
	GetRequestUserAgent() string

	SetAuthParameter(authMethodProtocol string, key string, value string, directKeyName ...bool)
	GetAuthParameter(authMethodProtocol string, key string, directKeyName ...bool) string
}

func extractToken(headerr string) string {
	fields := strings.Fields(headerr)
	if len(fields) != 2 || strings.ToLower(fields[0]) != AuthorizationBearer {
		return ""
	}
	return fields[1]
}

func SetAuthBearer(ctx AuthContext, value string) {
	ctx.SetAuthParameter("", AuthorizationHeader, utils.ConcatStrings(AuthorizationBearer, " ", value), true)
}

func GetAuthBearer(ctx AuthContext) string {
	authHeader := ctx.GetAuthParameter("", AuthorizationHeader, true)
	return extractToken(authHeader)
}

type AuthHandler interface {
	common.WithName

	Protocol() string

	Handle(ctx AuthContext) (bool, error)
	Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error
	Handlers() []AuthHandler
	SetAuthManager(manager AuthManager)

	ErrorDescriptions() map[string]string
	ErrorProtocolCodes() map[string]int
}

type AuthHandlerBase struct {
	common.WithNameBase

	protocol string
}

func (a *AuthHandlerBase) Init(protocol string) {
	a.protocol = protocol
	a.WithNameBase.Init(protocol)
}

func (a *AuthHandlerBase) ErrorDescriptions() map[string]string {
	return map[string]string{}
}

func (a *AuthHandlerBase) ErrorProtocolCodes() map[string]int {
	return map[string]int{}
}

func (a *AuthHandlerBase) Protocol() string {
	return a.protocol
}

func (a *AuthHandlerBase) Handlers() []AuthHandler {
	return nil
}

func (a *AuthHandlerBase) SetAuthManager(manager AuthManager) {
}
