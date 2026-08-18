package filestorage

import (
	"context"
	"testing"
)

// Task debug-sending-files-to-optimized-music (server stage 2 follow-up):
// regression test for a real crash - TempPath()/Path() checked tenancyCtx
// != nil but then called tenancyCtx.GetTenancy().GetID() unconditionally.
// GetTenancy() is nil whenever multitenancy is disabled (the shipped
// default) even when a non-nil TenancyContext is present on ctx, so this
// paniced the server on the very first real upload once stage 2's routing
// fix let a request actually reach UploadEndpoint.HandleRequest ->
// FileDataControllerBase.UploadPart -> FilesystemStorage.UploadPart ->
// TempPath() for the first time.
func TestTempPathAndPathNoTenancyDoesNotPanic(t *testing.T) {

	f := &FilesystemStorage{}
	f.TEMP_DIR = t.TempDir()
	f.BASE_DIR = t.TempDir()
	f.TENANCY_SUBFOLDER = "tenancy"
	f.TOPIC_SUBFOLDER = "topic"

	info := &testFileInfo{size: 10, topic: "topic1"}
	info.InitObject()

	ctx := context.Background()

	tempPath := f.TempPath(ctx, info, 0)
	if tempPath == "" {
		t.Fatal("TempPath returned an empty path")
	}

	path := f.Path(ctx, info)
	if path == "" {
		t.Fatal("Path returned an empty path")
	}
}
