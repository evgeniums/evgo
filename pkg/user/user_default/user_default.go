package user_default

import (
	"github.com/evgeniums/evgo/pkg/user"
)

type User struct {
	user.UserBase
}

type Users struct {
	user.UsersBase[*User]
}

func NewUser() *User {
	return &User{}
}

func NewOplog() user.OpLogUserI {
	return &user.OpLogUser{}
}

func NewUsers(userController ...user.UserController[*User]) *Users {
	m := &Users{}
	if len(userController) == 0 {
		c := user.LocalUserController[*User]()
		m.UsersBase.Construct(c)
		c.SetUserValidators(m)
	} else {
		m.UsersBase.Construct(userController[0])
	}
	m.UsersBase.SetUserBuilder(NewUser)
	return m
}
