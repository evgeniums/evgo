package sms_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/sms/sms_provider_factory"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/evgeniums/evgo/pkg/user"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _, testBasePath, _, _ = runtime.Caller(0)
var testDir = filepath.Dir(testBasePath)

func dbModels() []interface{} {
	return append([]interface{}{}, &sms.SmsMessage{})
}

func initSmsManager(t *testing.T, config ...string) (app_context.Context, sms.SmsManager) {
	app, _ := test_utils.InitAppContext(t, testDir, dbModels(), utils.OptionalArg("sms_test.json", config...))

	manager := sms.NewSmsManager()
	require.NoErrorf(t, manager.Init(app.Cfg(), app.Logger(), app.Validator(), &sms_provider_factory.MockFactory{}, "sms"), "failed to init SMS manager")

	return app, manager
}

func TestInitSmsManager(t *testing.T) {
	app, _ := initSmsManager(t)
	defer app.Close()
}

func testSms(t *testing.T, app app_context.Context, manager sms.SmsManager, encrypted bool) {

	user1 := user.NewUser()
	user1.InitObject()
	user1.LOGIN = "test_login1"
	user1.PHONE = "555000111"
	ctx1, sctx1 := test_utils.UserOpContext(app, "TestSendSms", user1)
	message1 := "Hello world, user1"
	smsId1, err := manager.Send(sctx1, message1, user1.PHONE)
	assert.NoError(t, err, "failed to send SMS")

	sms1, err := manager.FindSms(sctx1, smsId1)
	assert.NoError(t, err, "failed to find SMS")
	require.NotNil(t, sms1)
	assert.Equal(t, "mock_default", sms1.Provider)
	assert.Equal(t, sms.StatusSuccess, sms1.Status)

	if !encrypted {
		assert.Equal(t, message1, sms1.Message)
	} else {
		assert.NotEqual(t, message1, sms1.Message)
		m := manager.(*sms.SmsManagerBase)
		msg1, err := crypt_utils.DecryptStrings(m.SECRET, m.SALT, sms1.Message)
		assert.NoError(t, err, "failed to decrypt message")
		assert.Equal(t, message1, string(msg1))
	}

	ctx1.Close(sctx1)

	user2 := user.NewUser()
	user2.InitObject()
	user2.LOGIN = "test_login2"
	user2.PHONE = "999000111"
	ctx2, sctx2 := test_utils.UserOpContext(app, "TestSendSms", user2)
	message2 := "Hello world, user2"
	smsId2, err := manager.Send(sctx2, message2, user2.PHONE)
	assert.Error(t, err, "must fail sending SMS")

	sms2, err := manager.FindSms(sctx2, smsId2)
	assert.NoError(t, err, "failed to find SMS")
	require.NotNil(t, sms2)
	assert.Equal(t, "mock_fail", sms2.Provider)
	assert.Equal(t, sms.StatusFail, sms2.Status)
	ctx2.Close(sctx2)

	user3 := user.NewUser()
	user3.InitObject()
	user3.LOGIN = "test_login3"
	user3.PHONE = "990000111"
	ctx3, sctx3 := test_utils.UserOpContext(app, "TestSendSms", user3)
	message3 := "Hello world, user3"
	smsId3, err := manager.Send(sctx3, message3, user3.PHONE)
	assert.NoError(t, err, "failed to send SMS")

	sms3, err := manager.FindSms(sctx3, smsId3)
	assert.NoError(t, err, "failed to find SMS")
	require.NotNil(t, sms3)
	assert.Equal(t, "mock_success", sms3.Provider)
	assert.Equal(t, sms.StatusSuccess, sms3.Status)
	ctx3.Close(sctx3)
}

func TestSendSms(t *testing.T) {
	app, manager := initSmsManager(t)
	defer app.Close()
	testSms(t, app, manager, false)
}

func TestSendSmsEncrypted(t *testing.T) {
	app, manager := initSmsManager(t, "sms_encrypt_test.json")
	defer app.Close()
	testSms(t, app, manager, true)
}
