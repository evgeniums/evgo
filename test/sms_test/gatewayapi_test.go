package sms_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/evgeniums/evgo/pkg/app_context/app_default"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/sms/providers/gatewayapi"
	"github.com/evgeniums/evgo/pkg/test_utils"
)

func TestGatewayapi(t *testing.T) {

	configFile := "gatewayapi_config.json"

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

	sender := gatewayapi.New()
	err = sender.Init(app.Cfg(), app.Logger(), app.Validator())
	if err == nil {
		t.Fatalf("expected failure, got passed")
	}
	app.Logger().CheckFatalStack(app.Logger())

	err = sender.Init(app.Cfg(), app.Logger(), app.Validator(), "gatewayapi")
	if err != nil {
		app.Logger().CheckFatalStack(app.Logger())
		t.Fatalf("failed to init gatewayapi module")
	}

	opCtx := default_op_context.NewInitContext(app, app.Logger(), app.Db())
	sctx := op_context.MakeOpContext(opCtx)
	errManager := &generic_error.ErrorManagerBase{}
	errManager.Init(http.StatusBadRequest)
	opCtx.SetErrorManager(errManager)

	resp, err := sender.Send(sctx, "Confirmation code 9350", phone)
	opCtx.Close(sctx)
	if resp != nil {
		t.Logf("Response: %+v", resp)
	}
	if err != nil {
		t.Fatalf("failed to send SMS: %s", err)
	}
}
