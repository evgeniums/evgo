package user

import "github.com/evgeniums/evgo/pkg/oplog"

type OpLogUserI interface {
	oplog.Oplog
	Login() string
	UserId() string
	SetLogin(string)
	SetUserId(string)
}

type OpLogUserHolder struct {
	Login  string `gorm:"index" json:"login"`
	UserId string `gorm:"index" json:"user_id"`
}

type OpLogUser struct {
	oplog.OplogBase
	OpLogUserHolder
}

func (o *OpLogUser) SetLogin(val string) {
	o.OpLogUserHolder.Login = val
}

func (o *OpLogUser) Login() string {
	return o.OpLogUserHolder.Login
}

func (o *OpLogUser) SetUserId(val string) {
	o.OpLogUserHolder.UserId = val
}

func (o *OpLogUser) UserId() string {
	return o.OpLogUserHolder.UserId
}
