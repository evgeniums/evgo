package customer

import "github.com/evgeniums/evgo/pkg/user"

type OpLogCustomer struct {
	user.OpLogUser
}

func NewOplog() user.OpLogUserI {
	return &OpLogCustomer{}
}
