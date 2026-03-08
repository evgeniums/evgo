package access_control

import (
	"context"

	"github.com/evgeniums/evgo/pkg/common"
)

type Resource interface {
	common.WithNameAndPath
	IsOwner(subject Subject) bool
	OwnerAccess() Access
}

type ResourceManager interface {
	FindResource(sctx context.Context, path string) (Resource, error)
	ResourceTags(sctx context.Context, path string) ([]string, error)
}

type ResourceBase struct {
	common.WithNameAndPathBase
}
