package filedata_service

import (
	"context"
	"strconv"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/filestorage"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type FileInfoRegistry interface {
	Registry() filestorage.FileInfoRegistry
	UrlManager() filestorage.UrlManager
}

type FileInfoRegistryBase struct {
	registry   filestorage.FileInfoRegistry
	urlManager filestorage.UrlManager
}

func NewFileInfoRegistry(registry filestorage.FileInfoRegistry,
	urlManager filestorage.UrlManager) *FileInfoRegistryBase {

	f := &FileInfoRegistryBase{
		registry:   registry,
		urlManager: urlManager,
	}
	return f
}

func (f *FileInfoRegistryBase) Registry() filestorage.FileInfoRegistry {
	return f.registry
}

func (f *FileInfoRegistryBase) UrlManager() filestorage.UrlManager {
	return f.urlManager
}

func (f *FileInfoRegistryBase) FindForUpload(sctx context.Context, ids api.ResourceIds) (*filestorage.FileInfo, int64, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("FileInfoRegistry.FindForUpload")
	defer ctx.TraceOutMethod()

	id := ids.GetId(f.urlManager.IdUrlPathParameter())
	if id.Value() == "" {
		c.SetMessage("invalid file ID")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeBadRequest)
		return nil, 0, c.SetError(ctx.GenericError())
	}

	var part int64
	partStr := ids.GetId(f.urlManager.PartUrlPathParameter())
	if partStr.Value() != "" {
		var err error
		part, err = strconv.ParseInt(partStr.Value(), 10, 64)
		if err != nil {
			c.SetMessage("invalid file part")
			ctx.SetGenericErrorCode(generic_error.ErrorCodeBadRequest)
			return nil, 0, c.SetError(err)
		}
	}

	var topic string
	if f.urlManager.IsTopicEnabled() {
		topic = ids.GetId(f.urlManager.TopicUrlParameter()).Value()
	}

	info, err := f.registry.FindForUpload(sctx, id.Value(), part, topic)
	if err != nil {
		return nil, 0, c.SetError(err)
	}

	return info, part, nil
}

func (f *FileInfoRegistryBase) FindForDownload(sctx context.Context, ids api.ResourceIds) (*filestorage.FileInfo, error) {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("FileInfoRegistry.FindForDownload")
	defer ctx.TraceOutMethod()

	id := ids.GetId(f.urlManager.IdUrlPathParameter())
	if id.Value() == "" {
		c.SetMessage("invalid file ID")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeBadRequest)
		return nil, c.SetError(ctx.GenericError())
	}

	var topic string
	if f.urlManager.IsTopicEnabled() {
		topic = ids.GetId(f.urlManager.TopicUrlParameter()).Value()
	}

	info, err := f.registry.FindForDownload(sctx, id.Value(), topic)
	if err != nil {
		return nil, c.SetError(err)
	}

	return info, nil
}
