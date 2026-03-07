package crypt_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/stretchr/testify/assert"
)

func TestSha256(t *testing.T) {

	hash := crypt_utils.H256Hex([]byte(`{"field1":"value1","field2":100}`), "POST", "/service/create-object")
	assert.Equal(t, "eeda6be843c017e5bba65616463e09b8736f97cfac206529cdeb0466c4587b72", hash)

	hash = crypt_utils.H256B64([]byte("AVast5zVNKoVJoPQ"), "12345678")
	assert.Equal(t, "USX0DFXfMu6bQLE26Mbdx/B+7G15lf+YID74+ZKtY5A=", hash)
}
