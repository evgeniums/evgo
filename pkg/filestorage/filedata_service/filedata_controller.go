package filedata_service

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/api/api_server/rest_api_gin_server"
	"github.com/evgeniums/evgo/pkg/filestorage"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type FileDataController interface {
	FileInfoRegistry
	UploadPart(ctx context.Context) error
	Fetch(ctx context.Context) error
}

type FileDataControllerBase struct {
	FileInfoRegistryBase
	filestorageManager filestorage.StorageManager
	signedUrlHandler   filestorage.SignedUrlHandler
}

func NewFileDataController(registry filestorage.FileInfoRegistry,
	urlManager filestorage.UrlManager,
	filestorageManager filestorage.StorageManager,
	signedUrlHandler filestorage.SignedUrlHandler,
) *FileDataControllerBase {

	f := &FileDataControllerBase{
		FileInfoRegistryBase: FileInfoRegistryBase{
			registry:   registry,
			urlManager: urlManager,
		},
		filestorageManager: filestorageManager,
		signedUrlHandler:   signedUrlHandler,
	}

	return f
}

func (f *FileDataControllerBase) checkUrl(ctx context.Context, request api_server.Request, c op_context.CallContext, download bool) error {

	r, ok := request.(*rest_api_gin_server.Request)
	if !ok {
		return c.SetErrorStr("invalid request type, must be rest_api_gin_server.Request")
	}

	v := filestorage.SignUrlValues{Method: r.GetRequestMethod()}
	if !download {
		v.ContentLength = strconv.FormatInt(r.ContentLength(), 10)
	}
	err := f.signedUrlHandler.CheckUrl(ctx, r.GetGinCtx().Request.URL, &v)
	if err != nil {
		c.SetMessage("invalid URL")
		if err.Error() == "expired" {
			request.SetGenericErrorCode(generic_error.ErrorCodeExpired)
		} else {
			request.SetGenericErrorCode(generic_error.ErrorCodeBadRequest)
		}
		return c.SetError(err)
	}

	return nil
}

func (f *FileDataControllerBase) UploadPart(ctx context.Context) error {

	request := op_context.OpContext[api_server.Request](ctx)
	c := request.TraceInMethod("FileDataController.UploadPart")
	defer request.TraceOutMethod()

	err := f.checkUrl(ctx, request, c, false)
	if err != nil {
		return err
	}

	info, part, err := f.FindForUpload(ctx, request.ResourceIds())
	if err != nil {
		return c.SetError(err)
	}

	err = f.filestorageManager.UploadPart(ctx, info, request.UploadedData(), part)
	if err != nil {
		c.SetMessage("failed to upload part")
		return c.SetError(err)
	}

	return nil
}

func (f *FileDataControllerBase) Fetch(ctx context.Context) error {

	request := op_context.OpContext[api_server.Request](ctx)
	c := request.TraceInMethod("FileDataController.Fetch")
	defer request.TraceOutMethod()

	err := f.checkUrl(ctx, request, c, true)
	if err != nil {
		return err
	}

	info, err := f.FindForDownload(ctx, request.ResourceIds())
	if err != nil {
		return c.SetError(err)
	}

	rangeHeader := request.GetRequestHeader("Range")

	// fetch data
	var source io.ReadCloser
	var offset int64
	var responseHeaders map[string]string
	length := info.GetSize()
	if rangeHeader == "" {
		source, err = f.filestorageManager.Fetch(ctx, info)
		responseHeaders = map[string]string{
			"Accept-Ranges":       "bytes",
			"Content-Disposition": fmt.Sprintf(`attachment; filename="%s`, info.GetFileName()),
		}
	} else {

		offset, length, err = ParseRange(rangeHeader, info.GetSize())
		if err != nil {
			c.SetMessage("invalid range header")
			request.SetGenericErrorCode(generic_error.ErrorCodeBadRequest)
			return c.SetError(err)
		}

		source, err = f.filestorageManager.FetchRange(ctx, info, offset, length)
		if err == nil {
			endByte := offset + length - 1
			responseHeaders = map[string]string{
				"Content-Range":       fmt.Sprintf("bytes %d-%d/%d", offset, endByte, info.GetSize()),
				"Accept-Ranges":       "bytes",
				"Content-Disposition": fmt.Sprintf(`attachment; filename="%s`, info.GetFileName()),
			}
		}
	}
	if err != nil {
		c.SetMessage("failed to fetch data")
		if os.IsNotExist(err) {
			request.SetGenericErrorCode(generic_error.ErrorCodeNotFound)
		}
		return c.SetError(err)
	}

	// fill response
	request.Response().SetDataSource(source, length, info.GetContentType(), responseHeaders)

	// done
	return nil
}

func ParseRange(rangeHeader string, totalSize int64) (offset int64, length int64, err error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range prefix")
	}

	// Remove "bytes=" and split by "-"
	spec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(spec, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	startStr, endStr := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	switch {
	case startStr == "" && endStr != "":
		// Format: bytes=-500 (Last 500 bytes)
		suffix, _ := strconv.ParseInt(endStr, 10, 64)
		offset = totalSize - suffix
		if offset < 0 {
			offset = 0
		}
		length = totalSize - offset

	case startStr != "" && endStr == "":
		// Format: bytes=500- (From 500 to end)
		offset, _ = strconv.ParseInt(startStr, 10, 64)
		length = totalSize - offset

	case startStr != "" && endStr != "":
		// Format: bytes=0-499 (Specific range)
		offset, _ = strconv.ParseInt(startStr, 10, 64)
		end, _ := strconv.ParseInt(endStr, 10, 64)
		if end >= totalSize {
			end = totalSize - 1
		}
		length = end - offset + 1

	default:
		return 0, 0, fmt.Errorf("unsupported range")
	}

	if offset < 0 || length <= 0 {
		return 0, 0, fmt.Errorf("invalid calculated range")
	}

	return offset, length, nil
}
