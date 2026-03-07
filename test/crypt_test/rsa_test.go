package crypt_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/stretchr/testify/assert"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

func TestLoadPrivateRsaKey(t *testing.T) {

	keyPath := test_utils.AssetsFilePath(testDir, "private.key")
	passphrase := "12345"

	signer := crypt_utils.NewRsaSigner()
	err := signer.LoadKeyFromFile(keyPath, passphrase)
	assert.NoError(t, err)
}
