package auth_service

import (
	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_server"
)

// Login endpoint is derived from no handler endpoint because all processing in performed in auth preprocessing.
type LoginEndpoint struct {
	api_server.ResourceEndpoint
	api_server.EndpointNoHandler
}

func NewLoginEndpoint() *LoginEndpoint {
	ep := &LoginEndpoint{}
	api_server.InitResourceEndpoint(ep, "login", "Login", access_control.Post)
	return ep
}

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

type AuthService struct {
	api_server.ServiceBase
}

func NewAuthService(multitenancy ...bool) *AuthService {
	s := &AuthService{}
	s.Init("auth", api.PackageName, multitenancy...)
	s.AddChildren(NewLoginEndpoint(), NewLogoutEndpoint(), NewRefreshEndpoint())
	return s
}
