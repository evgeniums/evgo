package access_control

import "context"

type Rule interface {
	Resource() Resource
	Role() Role
	Access() Access
	Tags() []string
}

type Acl interface {
	FindRule(sctx context.Context, resourcePath string, tag string, role Role) (Rule, error)
}
