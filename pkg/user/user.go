package user

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_sms"
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type User interface {
	common.Object
	auth.User
	auth_login_phash.User
	auth_sms.UserWithPhone

	SetLogin(login string)
	SetPhone(phone string)
	SetBlocked(val bool)

	Email() string
	SetEmail(email string)

	DbUser() interface{}

	ToCmd(password string) interface{}
}

// TODO Configure somewhere unique indexes for phone and login if required

type UserPhone struct {
	PHONE string `gorm:"index" json:"phone,omitempty" validate:"omitempty,phone" vmessage:"Invalid phone format"`
}

type UserEmail struct {
	EMAIL string `gorm:"index" json:"email,omitempty" validate:"omitempty,email" vmessage:"Invalid email format"`
}

type UserBlocked struct {
	BLOCKED bool `gorm:"index" json:"blocked"`
}

type UserBaseFields struct {
	UserPhone
	UserEmail
	UserBlocked
	LOGIN string `gorm:"uniqueIndex" json:"login"`
}

func (u *UserBaseFields) Display() string {
	return u.LOGIN
}

func (u *UserBaseFields) Login() string {
	return u.LOGIN
}

func (u *UserBaseFields) SetLogin(login string) {
	u.LOGIN = login
}

func (u *UserBaseFields) Phone() string {
	return u.PHONE
}

func (u *UserBaseFields) SetPhone(phone string) {
	u.PHONE = phone
}

func (u *UserBaseFields) IsBlocked() bool {
	return u.BLOCKED
}

func (u *UserBaseFields) SetBlocked(val bool) {
	u.BLOCKED = val
}

func (u *UserBaseFields) Email() string {
	return u.EMAIL
}

func (u *UserBaseFields) SetEmail(email string) {
	u.EMAIL = email
}

func (u *UserBaseFields) Topic() string {
	return ""
}

func (u *UserBaseFields) SetUserFields(ctx op_context.Context, user User) ([]CheckDuplicateField, error) {
	user.SetEmail(u.Email())
	user.SetPhone(u.Phone())
	user.SetBlocked(u.IsBlocked())

	dups := make([]CheckDuplicateField, 0, 3)
	if u.Email() != "" {
		dups = append(dups, CheckDuplicateField{"email", u.Email(), ErrorCodeDuplicateEmail})
	}
	if u.Phone() != "" {
		dups = append(dups, CheckDuplicateField{"phone", u.Phone(), ErrorCodeDuplicatePhone})
	}

	return dups, nil
}

type UserPlainPassword struct {
	PlainPassword string `gorm:"-:all" json:"password"`
}

type UserFieldsWithPassword struct {
	UserBaseFields
	UserPlainPassword
}

func (u *UserFieldsWithPassword) Password() string {
	return u.PlainPassword
}

func (u *UserFieldsWithPassword) SetPassword(password string) {
	u.PlainPassword = password
}

func NewUserFieldsWihPassword() *UserFieldsWithPassword {
	u := &UserFieldsWithPassword{}
	return u
}

type UserBaseDB struct {
	common.ObjectBase
	UserBaseFields
	auth_login_phash.UserBase
	api.ResponseBase
}

func (u *UserBaseDB) ToCmd(password string) interface{} {
	cmd := &UserFieldsWithPassword{}
	cmd.UserBaseFields = u.UserBaseFields
	cmd.SetPassword(password)
	return cmd
}

type UserBase struct {
	UserBaseDB
}

func NewUser() *UserBase {
	u := &UserBase{}
	return u
}

func (u *UserBase) DbUser() interface{} {
	return &u.UserBaseDB
}

type UserFieldsSetter[T User] interface {
	Login() string
	Password() string
	SetUserFields(ctx op_context.Context, user T) ([]CheckDuplicateField, error)
}

type CheckDuplicateField struct {
	Name      string
	Value     interface{}
	ErrorCode string
}

type SetUserFields[UserType interface{}] func(ctx op_context.Context, user UserType) ([]CheckDuplicateField, error)

type UserFieldsSetterBase[T User] struct {
	UserFieldsWithPassword
}

func Phone[UserType User](phone string, userSample ...UserType) SetUserFields[UserType] {
	return func(ctx op_context.Context, user UserType) ([]CheckDuplicateField, error) {
		user.SetPhone(phone)
		if phone != "" {
			return []CheckDuplicateField{{"phone", phone, ErrorCodeDuplicatePhone}}, nil
		}
		return nil, nil
	}
}

func Email[UserType User](email string, userSample ...UserType) SetUserFields[UserType] {
	return func(ctx op_context.Context, user UserType) ([]CheckDuplicateField, error) {
		user.SetEmail(email)
		if email != "" {
			return []CheckDuplicateField{{"email", email, ErrorCodeDuplicateEmail}}, nil
		}
		return nil, nil
	}
}

func FindByLogin(controller crud.CRUD, ctx op_context.Context, login string, user interface{}, dest ...interface{}) (bool, error) {
	return controller.Read(ctx, db.Fields{"login": login}, user, dest...)
}

func FindByLoginAndTopic(controller crud.CRUD, ctx op_context.Context, login string, topic string, user interface{}, dest ...interface{}) (bool, error) {
	return controller.Read(ctx, db.Fields{"login": login, "topic": topic}, user, dest...)
}
