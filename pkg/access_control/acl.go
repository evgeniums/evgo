package access_control

import "github.com/evgeniums/evgo/pkg/op_context"

type Rule interface {
	Resource() Resource
	Role() Role
	Access() Access
	Tags() []string
}

type Acl interface {
	FindRule(ctx op_context.Context, resourcePath string, tag string, role Role) (Rule, error)
}
