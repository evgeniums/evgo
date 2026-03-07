package plain_login_service

import (
	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_token"
	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

// Logout endpoint is derived from no handler endpoint because all processing in performed in auth preprocessing.
type LogoutEndpoint struct {
	api_server.ResourceEndpoint
	api_server.EndpointNoHandler
}

func NewLogoutEndpoint() *LogoutEndpoint {
	ep := &LogoutEndpoint{}
	api_server.InitResourceEndpoint(ep, "logout", "Logout", access_control.Post)
	return ep
}

// Refresh endpoint is derived from no handler endpoint because all processing in performed in auth preprocessing.
type RefreshEndpoint struct {
	api_server.ResourceEndpoint
	api_server.EndpointNoHandler
}

func NewRefreshEndpoint() *RefreshEndpoint {
	ep := &RefreshEndpoint{}
	api_server.InitResourceEndpoint(ep, "refresh", "Refresh", access_control.Post)
	return ep
}

type PlainLoginService struct {
	api_server.ServiceBase

	users        auth_session.WithUserSessionManager
	tokenHandler *auth_token.AuthNewTokenHandler
}

func (p *PlainLoginService) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	path := utils.OptionalArg("plain_login_service", configPath...)

	tokenPath := object_config.Key(path, "token")
	err := p.tokenHandler.Init(cfg, log, vld, tokenPath)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of token handler in plain login service", err)
	}

	return nil
}

func NewPlainLoginService(users auth_session.WithUserSessionManager) *PlainLoginService {
	s := &PlainLoginService{users: users}
	s.ErrorsExtenderBase.Init(auth_token.ErrorDescriptions, auth_token.ErrorProtocolCodes)
	s.tokenHandler = auth_token.NewNewToken(users)
	s.ServiceBase.Init("auth", api.PackageName)
	s.AddChildren(NewLoginEndpoint(s), NewLogoutEndpoint(), NewRefreshEndpoint())
	return s
}
