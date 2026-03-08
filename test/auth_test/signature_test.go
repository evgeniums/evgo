package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/signature"
	"github.com/evgeniums/evgo/pkg/signature/user_pubkey"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var pubkey1Path = filepath.Join(assetsDir, "pub1.pem")
var privkey1Path = filepath.Join(assetsDir, "priv1.pem")
var privkey2Path = filepath.Join(assetsDir, "priv2.pem")

func TestSignature(t *testing.T) {
	app, users, server, _, sctx := initOpTest(t, "sig_test.jsonc")
	defer app.Close()

	pubKeyBuilder := func() *UserPubKey { return &UserPubKey{} }
	pubkeyController := user_pubkey.NewPubkeyController(pubKeyBuilder, server.SignatureManager(), users)
	pubKeyFinder := func(sctx context.Context) (signature.UserWithPubkey, error) {
		return user_pubkey.FindUserPubKey(pubkeyController, sctx)
	}
	server.SignatureManager().SetUserKeyFinder(pubKeyFinder)

	// create user1
	login1 := "user1@example.com"
	password1 := "password1"
	user1, err := users.Add(sctx, login1, password1, user.Phone("12345678", &User{}), user.Email("user1@example.com", &User{}))
	require.NoErrorf(t, err, "failed to add user")
	require.NotNil(t, user1)

	// prepare client
	client := test_utils.PrepareHttpClient(t, test_utils.BBGinEngine(t, server))

	// prepare signer
	signer1 := crypt_utils.NewRsaSigner()
	err = signer1.LoadKeyFromFile(privkey1Path, "")
	require.NoError(t, err)

	// good login
	client.Login(login1, password1)

	// try echo without signature
	path := "/status/echo"
	t.Logf("No pubkey")
	cmd1 := &Cmd{Param1: "value1_1", Param2: "Long text Long text Long text Long text Long text Long text Long text Long text Long text Long text Long text"}
	resp := client.Post(path, cmd1)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: auth.ErrorCodeUnauthorized})

	// no pubkey for user
	t.Logf("No pubkey")
	resp = client.PostSigned(t, signer1, path, cmd1)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: user_pubkey.ErrorCodeActiveKeyNotFound, Message: "Active public key not found"})

	// add pubkey for user 1
	pubKey1, err := os.ReadFile(pubkey1Path)
	require.NoError(t, err)
	keyId, err := pubkeyController.AddPubKey(sctx, user1.GetID(), string(pubKey1))
	require.NoError(t, err)
	assert.NotEmpty(t, keyId)

	// good signature
	t.Logf("Good signature")
	resp = client.PostSigned(t, signer1, path, cmd1)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusOK})

	// invalid key
	t.Logf("Good signature")
	signer2 := crypt_utils.NewRsaSigner()
	err = signer2.LoadKeyFromFile(privkey2Path, "")
	require.NoError(t, err)
	resp = client.PostSigned(t, signer2, path, cmd1)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: signature.ErrorCodeInvalidSignature})

	// mismatched content
	t.Logf("Mismatched content")
	cmd2 := &Cmd{Param1: "value1_2", Param2: "value1_2"}
	content, err := json.Marshal(cmd2)
	require.NoError(t, err)
	sig, err := signer1.SignB64(content, http.MethodPost, path)
	h := map[string]string{"x-evgo-signature-signature": sig}
	require.NoError(t, err)
	resp = client.Post(path, cmd1, h)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: signature.ErrorCodeInvalidSignature})

	// mismatched path
	t.Logf("Mismatched path")
	content, err = json.Marshal(cmd1)
	require.NoError(t, err)
	sig, err = signer1.SignB64(content, http.MethodPost, "/status/csrf")
	h = map[string]string{"x-evgo-signature-signature": sig}
	require.NoError(t, err)
	resp = client.Post(path, cmd1, h)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: signature.ErrorCodeInvalidSignature})

	// mismatched method
	t.Logf("Mismatched method")
	sig, err = signer1.SignB64(content, http.MethodPut, path)
	h = map[string]string{"x-evgo-signature-signature": sig}
	require.NoError(t, err)
	resp = client.Post(path, cmd1, h)
	test_utils.CheckResponse(t, resp, &test_utils.Expected{HttpCode: http.StatusUnauthorized, Error: signature.ErrorCodeInvalidSignature})
}
