package auth_session

import (
	"context"

	"github.com/evgeniums/evgo/pkg/auth"
)

type UserValidators interface {
	ValidateLogin(login string) error
	ValidatePassword(password string) error
}

type AuthUserFinder interface {
	FindAuthUser(sctx context.Context, login string, parameters ...map[string]string) (auth.User, error)
	FillAuthUser(sctx context.Context, useExistingSessisonParams ...bool) error
}

type AuthUserManager interface {
	UserValidators
	AuthUserFinder
}

type WithAuthUserManager interface {
	AuthUserManager() AuthUserManager
}

type WithSessionManager interface {
	SessionManager() SessionController
}

type WithUserSessionManager interface {
	WithAuthUserManager
	WithSessionManager
}

type WithUserSessionManagerBase struct {
	authUsers AuthUserManager
	sessions  SessionController
}

func (w *WithUserSessionManagerBase) AuthUserManager() AuthUserManager {
	return w.authUsers
}

func (w *WithUserSessionManagerBase) SessionManager() SessionController {
	return w.sessions
}

func NewUserAndSessionManager(authUsers AuthUserManager, sessions SessionController) *WithUserSessionManagerBase {
	m := &WithUserSessionManagerBase{authUsers: authUsers, sessions: sessions}
	return m
}
