package filestorage

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/test_utils"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

// TestFilesystemStorageInitAssignsFallbackHelper is a regression test for a bug where Init()
// built a fallback UploadPartHelper whenever NewFilesystemStorage() was called with none, but
// never assigned it to u.helper (unlike UrlManagerBase.Init, which does `u.helper = helper`).
// A FilesystemStorage built this way - the exact NewFilesystemStorage() call whitemservergo's
// file_controller would make if it ever stopped passing its own helper - nil-panicked on the
// first UploadPart instead of falling back to the helper Init() had just constructed and
// validated. Exercises the real Init() path (config load + validation), not a hand-built struct
// like the rest of this package's tests, since the bug was specifically in that wiring.
func TestFilesystemStorageInitAssignsFallbackHelper(t *testing.T) {
	app := test_utils.InitAppContextNoDb(t, testDir)
	defer app.Close()

	baseDir := t.TempDir()
	tempDir := t.TempDir()
	app.Cfg().Set("file.base_dir", baseDir)
	app.Cfg().Set("file.temp_dir", tempDir)

	storage := NewFilesystemStorage() // no helper passed - Init() must build one itself
	if err := storage.Init(app, "file", "file"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if storage.helper == nil {
		t.Fatal("Init() built a fallback UploadPartHelper but did not assign it to storage.helper")
	}

	ctx := context.Background()
	info := newTestInfo(5)
	if err := storage.UploadPart(ctx, info, bytes.NewReader([]byte("hello")), 0); err != nil {
		t.Fatalf("UploadPart failed using the Init()-assigned fallback helper: %v", err)
	}
	if err := storage.FinalizeUpload(ctx, info, 1); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	rc, err := storage.Fetch(ctx, info)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading fetched data failed: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("fetched data = %q, want %q", got, "hello")
	}
}
