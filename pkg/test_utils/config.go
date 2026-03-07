package test_utils

import (
	"path/filepath"

	"github.com/evgeniums/evgo/pkg/utils"
)

var Testing = ""
var ExternalConfigPath = ""
var Phone = ""

func IsTesting() bool {
	return Testing == "true"
}

func ExternalConfigFilePath(path string) string {
	if ExternalConfigPath == "" {
		return path
	}
	return filepath.Join(ExternalConfigPath, path)
}

func ExternalConfigFileExists(path string) bool {
	return utils.FileExists(ExternalConfigFilePath(path))
}

func AssetsFilePath(testDir string, fileName string) string {
	return filepath.Join(testDir, "assets", fileName)
}
