package filestorage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type FilesystemStorageConfig struct {
	BASE_DIR          string `validate:"required,dir" vmessage:"Required base_dir missing"`
	TEMP_DIR          string `validate:"omitempty,dir" vmessage:"Invalid temp_dir"`
	TENANCY_SUBFOLDER string `default:"tenancy" validate:"omitempty,alphanum" vmessage:"Invalid tenancy_subfolder"`
	TOPIC_SUBFOLDER   string `default:"topic" validate:"omitempty,alphanum" vmessage:"Invalid topic_subfolder"`
}

type FilesystemStorage struct {
	FilesystemStorageConfig
	helper UploadPartHelper
}

func NewFilesystemStorage(helper ...UploadPartHelper) *FilesystemStorage {
	f := &FilesystemStorage{}
	if len(helper) > 0 {
		f.helper = helper[0]
	}
	return f
}

func (u *FilesystemStorage) Config() any {
	return &u.FilesystemStorageConfig
}

func (u *FilesystemStorage) Init(app app_context.Context, parentConfigPath string, configPath ...string) error {

	// load config
	path := utils.OptionalString(object_config.Key(parentConfigPath, "filesystem_storage"), configPath...)
	err := object_config.LoadLogValidateApp(app, u, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of filesystem storage", err)
	}

	// init helper
	if u.helper == nil {
		helper := NewUploadPartHelper()
		err = helper.Init(app, path, path)
		if err != nil {
			return err
		}
		u.helper = helper
	}

	// done
	return nil
}

func (f *FilesystemStorage) TempPath(ctx context.Context, info FileInfo, partIndex ...int64) string {

	var dir string
	var d utils.Date
	d.SetTime(info.GetCreatedAt())

	// Task debug-sending-files-to-optimized-music (server stage 2, follow-up):
	// tenancyCtx being non-nil does not mean tenancyCtx.GetTenancy() is - see
	// UrlManagerBase.TenancyPath()'s identical guard (url_manager.go). With
	// multitenancy disabled GetTenancy() is always nil, so the bare
	// `tenancyCtx != nil` check here paniced on .GetID() the first time this
	// code path was ever actually exercised (the upload route 404'd before
	// stage 2's routing fix, so UploadPart() had never really run in this
	// deployment until now).
	tenancyCtx := op_context.OpContext[multitenancy.TenancyContext](ctx)
	if tenancyCtx != nil && tenancyCtx.GetTenancy() != nil {
		if info.GetTopic() == "" {
			dir = filepath.Join(f.TEMP_DIR, d.AsNumber(), f.TENANCY_SUBFOLDER, tenancyCtx.GetTenancy().GetID())
		} else {
			dir = filepath.Join(f.TEMP_DIR, d.AsNumber(), f.TENANCY_SUBFOLDER, tenancyCtx.GetTenancy().GetID(), f.TOPIC_SUBFOLDER, info.GetTopic())
		}
	} else {
		if info.GetTopic() == "" {
			dir = filepath.Join(f.TEMP_DIR, d.AsNumber())
		} else {
			dir = filepath.Join(f.TEMP_DIR, d.AsNumber(), f.TOPIC_SUBFOLDER, info.GetTopic())
		}
	}

	if len(partIndex) != 0 {
		return filepath.Join(dir, d.AsNumber(), info.GetID(), strconv.FormatInt(partIndex[0], 10))
	}
	return filepath.Join(dir, d.AsNumber(), info.GetID())
}

func (f *FilesystemStorage) Path(ctx context.Context, info FileInfo) string {

	var dir string

	// See TempPath()'s identical comment - tenancyCtx != nil alone does not
	// guarantee tenancyCtx.GetTenancy() != nil.
	tenancyCtx := op_context.OpContext[multitenancy.TenancyContext](ctx)
	if tenancyCtx != nil && tenancyCtx.GetTenancy() != nil {
		if info.GetTopic() == "" {
			dir = filepath.Join(f.BASE_DIR, f.TENANCY_SUBFOLDER, tenancyCtx.GetTenancy().GetID())
		} else {
			dir = filepath.Join(f.BASE_DIR, f.TENANCY_SUBFOLDER, tenancyCtx.GetTenancy().GetID(), f.TOPIC_SUBFOLDER, info.GetTopic())
		}
	} else {
		if info.GetTopic() == "" {
			dir = f.BASE_DIR
		} else {
			dir = filepath.Join(f.BASE_DIR, f.TOPIC_SUBFOLDER, info.GetTopic())
		}
	}

	return filepath.Join(dir, info.GetID())
}

func (f *FilesystemStorage) StartUpload(ctx context.Context, info FileInfo) error {
	return os.MkdirAll(f.TempPath(ctx, info), 0755)
}

// tempSiblingPath builds a same-directory scratch path for writing-then-renaming into path,
// e.g. ".../<id>/0" -> ".../<id>/_0_<random>". Previously this was built as
// fmt.Sprintf("_%s_%s", path, id), which prepends "_" to the *whole* path string - turning
// ".../<id>/0" into "_.../<id>/0_<random>", a completely different (and never-created)
// directory tree, not a hidden sibling file. That made every UploadPart/FinalizeUpload rename
// fail with ENOENT - found while adding files2 session 11a's round-trip test.
func tempSiblingPath(path string) string {
	dir, base := filepath.Split(path)
	return filepath.Join(dir, fmt.Sprintf("_%s_%s", base, utils.GenerateID()))
}

func (f *FilesystemStorage) UploadPart(ctx context.Context, info FileInfo, source io.Reader, partIndex ...int64) error {

	// prepare path
	idx := utils.OptionalArg(0, partIndex...)
	path := f.TempPath(ctx, info, idx)
	uploadPath := tempSiblingPath(path)

	// ensure the part's parent directory exists. StartUpload is meant to do this once per
	// upload, but nothing calls it (see files2-status.md session 11a); MkdirAll is idempotent
	// so doing it here per part is a safe, self-contained fix.
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	// open target file
	dest, err := os.Create(uploadPath)
	if err != nil {
		return err
	}

	// defer removing closing and file in case of error
	defer func() {
		if err != nil {
			dest.Close()
			os.Remove(uploadPath)
		}
	}()

	// copy data
	partLength := f.helper.UploadPartLength(info, partIndex...)
	_, err = io.CopyN(dest, source, partLength)
	if err != nil {
		return err
	}

	// check if there is an (n + 1)th byte remaining
	buf := make([]byte, 1)
	_, err1 := source.Read(buf)
	if err1 == nil {
		err = fmt.Errorf("source exceeds part size")
		return err
	}
	if err1 != io.EOF {
		err = err1
		return err
	}

	// close
	err = dest.Close()
	if err != nil {
		return err
	}

	// rename file part
	err = os.Rename(uploadPath, path)
	if err != nil {
		return err
	}

	// done
	return nil
}

func (f *FilesystemStorage) FinalizeUpload(ctx context.Context, info FileInfo, partsCount ...int64) error {

	targetPath := f.Path(ctx, info)
	tmpTargetPath := tempSiblingPath(targetPath)

	// ensure the destination directory (BASE_DIR, optionally + tenancy/topic subfolders)
	// exists - nothing else creates it (BASE_DIR itself is required to pre-exist by config
	// validation, but its tenancy/topic subfolders are not).
	err := os.MkdirAll(filepath.Dir(targetPath), 0755)
	if err != nil {
		return err
	}

	// open target file
	dst, err := os.Create(tmpTargetPath)
	if err != nil {
		return err
	}

	// defer cleanup in case of error
	defer func() {
		dst.Close()
		if err != nil {
			os.Remove(tmpTargetPath)
		}
	}()

	// copy parts to final destination
	count := utils.OptionalArg(1, partsCount...)
	for i := range count {
		src, err1 := os.Open(f.TempPath(ctx, info, i))
		if err1 != nil {
			err = err1
			return err
		}
		_, err = io.Copy(dst, src)
		src.Close()
		if err != nil {
			return err
		}
	}

	// rename file
	err = dst.Close()
	if err != nil {
		return err
	}
	err = os.Rename(tmpTargetPath, targetPath)
	if err != nil {
		return err
	}

	// remove temporary files
	os.RemoveAll(f.TempPath(ctx, info))

	// done
	return nil
}

func (f *FilesystemStorage) Fetch(ctx context.Context, info FileInfo, offset ...int64) (io.ReadCloser, error) {

	path := f.Path(ctx, info)

	// open file
	src, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// move the internal cursor to the requested offset
	if _, err := src.Seek(utils.OptionalArg(0, offset...), io.SeekStart); err != nil {
		src.Close()
		return nil, err
	}

	// done
	return src, nil
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}

func (f *FilesystemStorage) FetchRange(ctx context.Context, info FileInfo, offset int64, length int64) (io.ReadCloser, error) {

	readCloser, err := f.Fetch(ctx, info, offset)
	if err != nil {
		return nil, err
	}

	return &limitedReadCloser{
		Reader: io.LimitReader(readCloser, length),
		Closer: readCloser,
	}, nil
}

func (f *FilesystemStorage) Delete(ctx context.Context, pathPrefix string) error {
	path := filepath.Join(f.BASE_DIR, pathPrefix)
	return os.RemoveAll(path)
}

func (f *FilesystemStorage) DeleteFile(ctx context.Context, info FileInfo) error {
	// os.RemoveAll on a path that does not exist returns nil, so this is safe to call for
	// content that was never finalized (e.g. a TTL firing on an abandoned upload).
	return os.RemoveAll(f.Path(ctx, info))
}

func (f *FilesystemStorage) DeleteTemp(ctx context.Context, toDate utils.Date) error {

	minName := toDate.AsNumber()

	entries, _ := os.ReadDir(f.TEMP_DIR)
	for _, entry := range entries {
		if entry.Name() < minName {
			fullPath := filepath.Join(f.TEMP_DIR, entry.Name())
			err := os.RemoveAll(fullPath)
			if err != nil {
				fmt.Printf("Failed to delete %s: %v\n", fullPath, err)
				continue
			}
		}
	}

	return nil
}
