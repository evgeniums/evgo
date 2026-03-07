package admin

import "github.com/evgeniums/evgo/pkg/user"

type OpLogAdmin struct {
	user.OpLogUser
}

func NewOplog() user.OpLogUserI {
	return &OpLogAdmin{}
}
