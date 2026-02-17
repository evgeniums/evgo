package auth

import (
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/op_context/default_op_context"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type User interface {
	GetID() string
	Display() string
	Login() string
	IsBlocked() bool
}

type UserBase struct {
	UserId      string `gorm:"index"`
	UserLogin   string `gorm:"index"`
	UserDisplay string `gorm:"index"`
	UserBlocked bool   `gorm:"index"`
}

func (u *UserBase) GetID() string {
	return u.UserId
}

func (u *UserBase) Display() string {
	return u.UserDisplay
}

func (u *UserBase) Login() string {
	return u.UserLogin
}

func (u *UserBase) IsBlocked() bool {
	return u.UserBlocked
}

func NewAuthUser(id string, login string, display string, blocked ...bool) *UserBase {

	userLogin := login
	if userLogin == "" {
		userLogin = id
	}
	userDisplay := display
	if userDisplay == "" {
		userDisplay = userLogin
	}

	return &UserBase{UserId: id, UserLogin: userLogin, UserDisplay: userDisplay, UserBlocked: utils.OptionalArg(false, blocked...)}
}

type WithUser interface {
	SetUser(user User)
	GetUserId() string
	GetUserDisplay() string
	GetUserLogin() string
}

type WithUserBase struct {
	UserId      string `gorm:"index"`
	UserLogin   string `gorm:"index"`
	UserDisplay string `gorm:"index"`
}

func (w *WithUserBase) SetUser(user User) {
	w.UserId = user.GetID()
	w.UserLogin = user.Login()
	w.UserDisplay = user.Display()
}

func (s *WithUserBase) GetUserId() string {
	return s.UserId
}

func (s *WithUserBase) GetUserDisplay() string {
	return s.UserDisplay
}

func (s *WithUserBase) GetUserLogin() string {
	return s.UserLogin
}

type Session interface {
	GetSessionId() string
	SetSessionId(id string)
	GetClientId() string
	SetClientId(id string)
	IsLoggedIn() bool
}

type SessionBase struct {
	session string
	client  string
}

func (u *SessionBase) GetSessionId() string {
	return u.session
}

func (u *SessionBase) SetSessionId(id string) {
	u.session = id
}

func (u *SessionBase) GetClientId() string {
	return u.client
}

func (u *SessionBase) SetClientId(id string) {
	u.client = id
}

func (u *SessionBase) IsLoggedIn() bool {
	return u.session != ""
}

type WithAuthUser interface {
	AuthUser() User
	SetAuthUser(user User)
}

type ContextWithAuthUser interface {
	op_context.Context
	WithAuthUser
}

type UserContext interface {
	multitenancy.TenancyContext
	WithAuthUser
}

func Tenancy(ctx UserContext) string {
	return multitenancy.ContextTenancyId(ctx)
}

type TenancyUserContext struct {
	multitenancy.TenancyContextBase
	User User
}

func (u *TenancyUserContext) AuthUser() User {
	return u.User
}

func (u *TenancyUserContext) SetAuthUser(user User) {
	u.User = user
	u.SetLoggerField("user", AuthUserDisplay(u))
}

type UserContextBase struct {
	op_context.Context
	User User
}

func NewUserContext(fromCtx ...op_context.Context) *UserContextBase {
	c := &UserContextBase{}
	if len(fromCtx) == 0 {
		c.Context = default_op_context.NewContext()
	} else {
		c.Context = fromCtx[0]
	}
	return c
}

func (u *UserContextBase) AuthUser() User {
	return u.User
}

func (u *UserContextBase) SetAuthUser(user User) {
	u.User = user
	u.SetLoggerField("user", AuthUserDisplay(u))
}

func AuthUserDisplay(ctx WithAuthUser) string {
	if ctx != nil {
		u := ctx.AuthUser()
		if u != nil {
			if u.Display() != "" {
				return u.Display()
			}
			if u.Login() != "" {
				return u.Login()
			}
			if u.GetID() != "" {
				return u.GetID()
			}
		}
	}
	return ""
}
