package oplog

import (
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/db"
)

type Oplog interface {
	common.Object
	Operation() string
	SetOperation(val string)
	Context() string
	SetContext(val string)
	ContextName() string
	SetContextName(val string)
	OriginApp() string
	SetOriginApp(val string)
	OriginName() string
	SetOriginName(val string)
	OriginSource() string
	SetOriginSource(val string)
	OriginClient() string
	SetOriginClient(string)
	User() string
	SetUser(string)
	UserType() string
	SetUserType(string)
}

type OplogHolder struct {
	Operation    string `gorm:"index" json:"operation"`
	Context      string `gorm:"index" json:"context"`
	ContextName  string `gorm:"index" json:"context_name"`
	OriginApp    string `gorm:"index" json:"origin_app"`
	OriginName   string `gorm:"index" json:"origin_name"`
	User         string `gorm:"index" json:"origin_user"`
	OriginSource string `gorm:"index" json:"origin_source"`
	OriginClient string `gorm:"index" json:"origin_client"`
	UserType     string `gorm:"index" json:"origin_user_type"`
}

type OplogBase struct {
	common.ObjectBase
	OplogHolder
}

func (o *OplogBase) Operation() string {
	return o.OplogHolder.Operation
}

func (o *OplogBase) SetOperation(val string) {
	o.OplogHolder.Operation = val
}

func (o *OplogBase) Context() string {
	return o.OplogHolder.Context
}

func (o *OplogBase) SetContext(val string) {
	o.OplogHolder.Context = val
}

func (o *OplogBase) ContextName() string {
	return o.OplogHolder.ContextName
}

func (o *OplogBase) SetContextName(val string) {
	o.OplogHolder.ContextName = val
}

func (o *OplogBase) OriginApp() string {
	return o.OplogHolder.OriginApp
}

func (o *OplogBase) SetOriginApp(val string) {
	o.OplogHolder.OriginApp = val
}

func (o *OplogBase) OriginName() string {
	return o.OplogHolder.OriginName
}

func (o *OplogBase) SetOriginName(val string) {
	o.OplogHolder.OriginName = val
}

func (o *OplogBase) OriginSource() string {
	return o.OplogHolder.OriginSource
}

func (o *OplogBase) SetOriginSource(val string) {
	o.OplogHolder.OriginSource = val
}

func (o *OplogBase) OriginClient() string {
	return o.OplogHolder.OriginClient
}

func (o *OplogBase) SetOriginClient(val string) {
	o.OplogHolder.OriginClient = val
}

func (o *OplogBase) User() string {
	return o.OplogHolder.User
}

func (o *OplogBase) SetUser(val string) {
	o.OplogHolder.User = val
}

func (o *OplogBase) UserType() string {
	return o.OplogHolder.UserType
}

func (o *OplogBase) SetUserType(val string) {
	o.OplogHolder.UserType = val
}

type OplogController interface {
	Write(o Oplog) error
	Read(filter *db.Filter, docs interface{}) (int64, error)
}
