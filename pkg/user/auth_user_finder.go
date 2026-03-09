package user

import (
	"context"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/crud"
)

type AuthUserFinderBase struct {
	crud.WithCRUDBase
	userBuilder func() User
}

func (a *AuthUserFinderBase) FindAuthUser(sctx context.Context, login string, parameters ...map[string]string) (auth.User, error) {
	user := a.userBuilder()
	var found bool
	var err error
	found, err = FindByLogin(a.CRUD(), sctx, login, user)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return user, nil
}

func (a *AuthUserFinderBase) FillAuthUser(sctx context.Context, useExistingSessisonParams ...bool) error {
	return nil
}

func NewAuthUserFinder(userBuilder func() User, cruds ...crud.CRUD) *AuthUserFinderBase {
	a := &AuthUserFinderBase{userBuilder: userBuilder}
	a.Construct(cruds...)
	return a
}
