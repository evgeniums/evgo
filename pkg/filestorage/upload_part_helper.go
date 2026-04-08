package filestorage

import (
	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/utils"
)

type UploadPartHelperConfig struct {
	UPLOAD_PART_LENGTH int64 `default:"8388608"`
}

type UploadPartHelperBase struct {
	UploadPartHelperConfig
}

func NewUploadPartHelper() *UploadPartHelperBase {
	u := &UploadPartHelperBase{}
	return u
}

func (s *UploadPartHelperBase) Config() any {
	return &s.UploadPartHelperConfig
}

func (s *UploadPartHelperBase) Init(app app_context.Context, parentConfigPath string, configPath ...string) error {

	path := utils.OptionalString(object_config.Key(parentConfigPath, "upload_part_helper"), configPath...)
	err := object_config.LoadLogValidateApp(app, s, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of upload part helper", err)
	}

	return nil
}

func (w *UploadPartHelperConfig) UploadPartLength(info FileInfo, partIndex ...int64) int64 {
	if len(partIndex) == 0 {
		return info.GetSize()
	}

	l := info.GetUploadPartSize()
	if l == 0 {
		l = w.UPLOAD_PART_LENGTH
	}

	return FilePartLength(info.GetSize(), l, partIndex...)
}

func (w *UploadPartHelperConfig) PartCount(info FileInfo) int64 {

	maxPartSize := info.GetUploadPartSize()
	if maxPartSize == 0 {
		maxPartSize = w.UPLOAD_PART_LENGTH
	}

	if info.GetSize() <= 0 || maxPartSize <= 0 {
		return 0
	}

	return (info.GetSize() + maxPartSize - 1) / maxPartSize
}

func FilePartLength(totalSize int64, maxPartSize int64, partIndex ...int64) int64 {
	// If no index is provided, return the total size
	if len(partIndex) == 0 {
		return totalSize
	}

	index := partIndex[0]
	start := index * maxPartSize

	// If the requested index starts beyond the file size, length is 0
	if start >= totalSize {
		return 0
	}

	// Calculate remaining bytes from this start point
	remaining := totalSize - start

	// If remaining bytes are more than a full part, return maxPartSize
	if remaining > maxPartSize {
		return maxPartSize
	}

	// Otherwise, return the last (smaller) part length
	return remaining
}
