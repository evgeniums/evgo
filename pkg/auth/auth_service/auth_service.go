package auth_service

import (
	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
)

// Negotiate endpoint is derived from no handler endpoint because all processing in performed in auth preprocessing.
type NegotiateEndpoint struct {
	api_server.ResourceEndpoint
	api_server.EndpointNoHandler
}

func NewNegotiateEndpoint() *NegotiateEndpoint {
	ep := &NegotiateEndpoint{}
	api_server.InitResourceEndpoint(ep, "negotiate", "Negotiate", access_control.Post)
	return ep
}

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

	negEp     *NegotiateEndpoint
	loginEp   *LoginEndpoint
	logoutEp  *LogoutEndpoint
	refreshEp *RefreshEndpoint
}

func NewAuthService(multitenancy ...bool) *AuthService {
	s := &AuthService{}
	s.Init("auth", api.PackageName, multitenancy...)
	s.negEp = NewNegotiateEndpoint()
	s.loginEp = NewLoginEndpoint()
	s.logoutEp = NewLogoutEndpoint()
	s.refreshEp = NewRefreshEndpoint()
	s.AddChildren(s.negEp, s.loginEp, s.logoutEp, s.refreshEp)
	return s
}

func (a *AuthService) NegotiateEndpoint() *NegotiateEndpoint {
	return a.negEp
}

func (a *AuthService) LoginEndpoint() *LoginEndpoint {
	return a.loginEp
}

func (a *AuthService) LogoutEndpoint() *LogoutEndpoint {
	return a.logoutEp
}

func (a *AuthService) RefreshEndpoint() *RefreshEndpoint {
	return a.refreshEp
}
