package confirmation_control

import (
	"context"

	"github.com/evgeniums/evgo/pkg/generic_error"
)

const PackageName = "confirmation_control"

const StatusSuccess string = "success"
const StatusFailed string = "failed"
const StatusCancelled string = "cancelled"

type ConfirmationSender interface {
	SendConfirmation(sctx context.Context, operationId string, recipient string, failedUrl string, parameters ...map[string]interface{}) (redirectUrl string, err error)
}

type ConfirmationResult struct {
	Code   string                   `json:"code,omitempty"`
	Status string                   `json:"status,omitempty"`
	Error  *generic_error.ErrorBase `json:"error,omitempty"`
}

type ConfirmationCallbackHandler interface {
	ConfirmationCallback(sctx context.Context, operationId string, result *ConfirmationResult) (redirectUrl string, err error)
}
