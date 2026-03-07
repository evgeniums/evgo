package sms_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/sms/providers/smsru"
	"github.com/evgeniums/evgo/pkg/test_utils"
)

func TestSmsru(t *testing.T) {

	configFile := "smsru_config.json"

	fmt.Printf("config=%s, phone=%s\n", test_utils.ExternalConfigFilePath(configFile), test_utils.Phone)
	if !test_utils.ExternalConfigFileExists(configFile) || test_utils.Phone == "" {
		t.Skip("Skip TestGatewayapi because external config or phone not defined")
	}

	configFile = test_utils.ExternalConfigFilePath(configFile)
	phone := test_utils.Phone

	app := app_default.New(nil)
	err := app.Init(configFile)
	if err != nil {
		t.Fatalf("failed to init application context: %s", err)
	}

	sender := smsru.New()
	err = sender.Init(app.Cfg(), app.Logger(), app.Validator())
	if err == nil {
		t.Fatalf("expected failure, got passed")
	}
	app.Logger().CheckFatalStack(app.Logger())

	err = sender.Init(app.Cfg(), app.Logger(), app.Validator(), "smsru")
	if err != nil {
		app.Logger().CheckFatalStack(app.Logger())
		t.Fatalf("failed to init smsru module")
	}

	opCtx := default_op_context.NewContext()
	opCtx.Init(app, app.Logger(), app.Db())
	errManager := &generic_error.ErrorManagerBase{}
	errManager.Init(http.StatusBadRequest)
	opCtx.SetErrorManager(errManager)

	resp, err := sender.Send(opCtx, "Confirmation code 1015", phone)
	opCtx.Close()
	if resp != nil {
		t.Logf("Response: %+v", resp)
	}
	if err != nil {
		t.Fatalf("failed to send SMS: %s", err)
	}
}
