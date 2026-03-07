package utils_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/utils"
)

var (
	_, scriptPath, _, _ = runtime.Caller(0)
)

func TestWalkDirExt(t *testing.T) {

	handler := func(filePath string) error {
		t.Logf("File path: %s", filePath)
		return nil
	}

	root := filepath.Dir(scriptPath)
	utils.WalkDirExt(handler, root, "go")
	// utils.WalkDirExt(handler, root, "pem")
}
