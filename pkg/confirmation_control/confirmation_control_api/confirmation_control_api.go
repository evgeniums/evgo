package confirmation_control_api

import (
	"fmt"

	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/confirmation_control"
	"github.com/evgeniums/go-utils/pkg/generic_error"
)

const ServiceName string = "confirmation"

const OperationResource string = "operation"
const CallbackResource string = "callback"
const FailedResource string = "failed"

func CheckConfirmation() api.Operation {
	return api.Post("check_confirmation")
}

func PrepareCheckConfirmation() api.Operation {
	return api.Get("prepare_check_confirmation")
}

func PrepareOperation() api.Operation {
	return api.Post("prepare_operation")
}

func FailedConfirmation() api.Operation {
	return api.Post("failed_confirmation")
}

type PrepareOperationCmd struct {
	Id         string                 `json:"id" validate:"required,id" vmessage:"Operation ID must be specified"`
	Recipient  string                 `json:"recipient" validate:"required" vmessage:"Recipient must be specified"`
	FailedUrl  string                 `json:"failed_url" validate:"required,url" vmessage:"Invalid format of failed URL"`
	Parameters map[string]interface{} `json:"parameters"`
	Ttl        int                    `json:"ttl"`
}

type PrepareOperationResponse struct {
	api.ResponseStub
	Url string `json:"url"`
}

type CheckConfirmationResponse struct {
	api.ResponseStub
	RedirectUrl string `json:"redirect_url"`
}

type PrepareCheckConfirmationResponse struct {
	api.ResponseStub
	CodeInBody bool                   `json:"code_in_body"`
	FailedUrl  string                 `json:"failed_url"`
	Parameters map[string]interface{} `json:"parameters"`
}

type OperationCacheToken struct {
	Id         string                 `json:"id"`
	Recipient  string                 `json:"recipient"`
	FailedUrl  string                 `json:"failed_url"`
	Parameters map[string]interface{} `json:"parameters"`
}

const ConfirmationCacheKey = "confirmation_service"

func OperationIdCacheKey(operationId string) string {
	return fmt.Sprintf("%s/%s", ConfirmationCacheKey, operationId)
}

func GetTokenFromCache(ctx auth.AuthContext) (*OperationCacheToken, error) {

	// setup
	c := ctx.TraceInMethod("GetTokenFromCache")
	defer ctx.TraceOutMethod()

	// get token from cache
	operationId := ctx.GetResourceId(OperationResource)
	ctx.SetLoggerField("confirmation_id", operationId)
	cacheToken := &OperationCacheToken{}
	cacheKey := OperationIdCacheKey(operationId.Value())
	found, err := ctx.Cache().Get(cacheKey, cacheToken)
	if err != nil {
		c.SetMessage("failed to get cache token")
		ctx.SetGenericErrorCode(generic_error.ErrorCodeInternalServerError)
		return nil, c.SetError(err)
	}
	if !found {
		ctx.SetGenericErrorCode(generic_error.ErrorCodeExpired)
		return nil, c.SetErrorStr("cache token not found")
	}

	// done
	return cacheToken, nil
}

func DeleteTokenFromCache(ctx auth.AuthContext) {

	// setup
	c := ctx.TraceInMethod("DeleteTokenFromCache")
	defer ctx.TraceOutMethod()

	// get token from cache
	operationId := ctx.GetResourceId(OperationResource)
	ctx.SetLoggerField("confirmation_id", operationId)
	cacheKey := OperationIdCacheKey(operationId.Value())
	err := ctx.Cache().Unset(cacheKey)
	if err != nil {
		c.Logger().Warn("failed to delete cache token")
	}
}

func CallbackConfirmation() api.Operation {
	return api.Post("callback_confirmation")
}

type CallbackConfirmationCmd struct {
	Id string `json:"operation_id" validate:"required,id" vmessage:"Invalid operation ID"`
	confirmation_control.ConfirmationResult
}

type CallbackConfirmationResponse struct {
	api.ResponseStub
	Url string `json:"url"`
}
