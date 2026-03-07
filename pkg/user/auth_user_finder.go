package user

import (
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/crud"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type AuthUserFinderBase struct {
	crud.WithCRUDBase
	userBuilder func() User
}

func (a *AuthUserFinderBase) FindAuthUser(ctx op_context.Context, login string) (auth.User, error) {
	user := a.userBuilder()
	var found bool
	var err error
	found, err = FindByLogin(a.CRUD(), ctx, login, user)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return user, nil
}

func (a *AuthUserFinderBase) FillAuthUser(ctx op_context.Context, useExistingSessisonParams ...bool) error {
	return nil
}

func NewAuthUserFinder(userBuilder func() User, cruds ...crud.CRUD) *AuthUserFinderBase {
	a := &AuthUserFinderBase{userBuilder: userBuilder}
	a.Construct(cruds...)
	return a
}
