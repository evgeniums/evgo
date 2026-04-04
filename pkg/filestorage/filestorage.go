package filestorage

import (
	"context"
	"io"
	"net/url"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/utils"
)

type FileInfo struct {
	common.ObjectBase

	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	FileName    string `json:"fileName"`

	Topic string `json:"topic"`

	NativeId       string `json:"native_id"`
	UploadPartSize int64  `json:"upload_part_size"`
}

type SignedUrlHandler interface {
	SignUrl(ctx context.Context, originalUrl string, method string) (string, error)
	CheckUrlString(ctx context.Context, signedUrl string, method string) error

	CheckUrl(ctx context.Context, signedUrl *url.URL, method string) error
}

type UploadUrlInfo struct {
	Urls          []string
	TotalUrlCount int64
	Method        string
	FromPartIndex int64
}

type UploadPartHelper interface {
	UploadPartLength(info *FileInfo, partIndex ...int64) int64
	PartCount(info *FileInfo) int64
}

type UrlManager interface {
	GetUploadUrls(ctx context.Context, info *FileInfo, fromPartIndex ...int64) (*UploadUrlInfo, error)
	GetDownloadUrl(ctx context.Context, info *FileInfo) (string, error)

	IdUrlPathParameter() string
	PartUrlPathParameter() string
	TopicUrlParameter() string
	IsTopicEnabled() bool
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

type FileInfoRegistry interface {
	FindForUpload(ctx context.Context, id string, part int64, topic ...string) (*FileInfo, error)
	FindForDownload(ctx context.Context, id string, topic ...string) (*FileInfo, error)
}
