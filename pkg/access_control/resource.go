package access_control

import (
	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type Resource interface {
	common.WithNameAndPath
	IsOwner(subject Subject) bool
	OwnerAccess() Access
}

type ResourceManager interface {
	FindResource(ctx op_context.Context, path string) (Resource, error)
	ResourceTags(ctx op_context.Context, path string) ([]string, error)
}

type ResourceBase struct {
	common.WithNameAndPathBase
}
