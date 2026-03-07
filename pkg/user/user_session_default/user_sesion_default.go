package user_session_default

import (
	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/user/user_default"
)

type User = user_default.User

type UserSession struct {
	auth_session.SessionBase
}

func NewSession() *UserSession {
	return &UserSession{}
}

type UserSessionClient struct {
	auth_session.SessionClientBase
}

func NewSessionClient() *UserSessionClient {
	return &UserSessionClient{}
}

type Users = user.UsersWithSessionBase[*User, *UserSession, *UserSessionClient]

func NewUsers(controllers ...user.UsersWithSessionBaseConfig[*User]) *Users {
	return user.NewUsersWithSession(user_default.NewUser, NewSession, NewSessionClient, user_default.NewOplog, controllers...)
}
