package customer

import (
	"github.com/evgeniums/go-utils/pkg/auth/auth_session"
	"github.com/evgeniums/go-utils/pkg/common"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/user"
	"github.com/evgeniums/go-utils/pkg/user/user_session_default"
)

type User interface {
	user.User
	common.WithName
	common.WithDescription
	common.WithTitle
}

type UserBase struct {
	user_session_default.User
	common.WithNameBase
	common.WithDescriptionBase
	common.WithTitleBase
}

type Customer struct {
	UserBase
}

func NewCustomer() *Customer {
	c := &Customer{}
	return c
}

type CustomerSession struct {
	auth_session.SessionBase
}

func NewCustomerSession() *CustomerSession {
	return &CustomerSession{}
}

type CustomerSessionClient struct {
	user_session_default.UserSessionClient
}

func NewCustomerSessionClient() *CustomerSessionClient {
	return &CustomerSessionClient{}
}

func Name(name string, sample ...User) user.SetUserFields[User] {
	return func(ctx op_context.Context, user User) ([]user.CheckDuplicateField, error) {
		user.SetName(name)
		return nil, nil
	}
}

func Description(description string, sample ...User) user.SetUserFields[User] {
	return func(ctx op_context.Context, user User) ([]user.CheckDuplicateField, error) {
		user.SetDescription(description)
		return nil, nil
	}
}

func Title(title string, sample ...User) user.SetUserFields[User] {
	return func(ctx op_context.Context, user User) ([]user.CheckDuplicateField, error) {
		user.SetTitle(title)
		return nil, nil
	}
}
