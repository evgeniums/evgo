package admin

import (
	"github.com/evgeniums/evgo/pkg/auth/auth_session"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/user/user_session_default"
)

type Role struct {
	common.IDBase
	Name string
}

type Admin struct {
	user_session_default.User
	Roles []Role `gorm:"-:all" json:"roles,omitempty"`
}

func NewAdmin() *Admin {
	// TODO fill roles from database
	a := &Admin{}
	a.Roles = make([]Role, 1)
	a.Roles[0] = Role{Name: "superadmin"}
	return a
}

type AdminSession struct {
	auth_session.SessionBase
}

func NewAdminSession() *AdminSession {
	return &AdminSession{}
}

type AdminSessionClient struct {
	user_session_default.UserSessionClient
}

func NewAdminSessionClient() *AdminSessionClient {
	return &AdminSessionClient{}
}
