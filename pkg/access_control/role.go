package access_control

import "github.com/evgeniums/evgo/pkg/common"

type Role interface {
	common.WithName
}

type RoleBase struct {
	common.WithNameBase
}
