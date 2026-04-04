package filestorage

import (
	"context"
	"io"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/utils"
)

type FileInfo struct {
	common.ObjectBase

	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`

	NativeId       string `json:"native_id"`
	UploadPartSize int64  `json:"upload_part_size"`
}

type SignedUrlHandler interface {
	SignUrl(ctx context.Context, originalUrl string, method string) (string, error)
	CheckUrl(ctx context.Context, signedUrl string, method string) error
}

type UrlManager interface {
	GetUploadUrl(ctx context.Context, info *FileInfo, partIndex ...int64) (string, error)
	GetDownloadUrl(ctx context.Context, info *FileInfo, partIndex ...int64) (string, error)

	IsSingleUploadUrl() bool
	IsSingleDownloadUrl() bool
}

type StorageManager interface {
	StartUpload(ctx context.Context, info *FileInfo) error
	UploadPart(ctx context.Context, info *FileInfo, source io.Reader, partIndex ...int64) error
	FinalizeUpload(ctx context.Context, info *FileInfo, partsCount ...int64) error

	Fetch(ctx context.Context, info *FileInfo, offset ...int64) (io.ReadCloser, error)
	FetchRange(ctx context.Context, info *FileInfo, offset int64, length int64) (io.ReadCloser, error)

	Delete(ctx context.Context, pathPrefix string) error

	DeleteTemp(ctx context.Context, toDate utils.Date) error
}
