package api_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

type AuthSms struct {
	code  string
	token string
}

func (a *AuthSms) HandleResponse(resp OperationResponse) {
	if resp == nil {
		return
	}
	a.token = resp.GetHeader("X-Auth-Sms-Token")
}

func (a *AuthSms) SetCode(code string) {
	a.code = code
}

func (a *AuthSms) MakeHeaders(sctx context.Context, operation api.Operation, cmd interface{}) (map[string]string, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	ctx.TraceInMethod("AuthSms.MakeHeaders")
	defer ctx.TraceOutMethod()

	// no headers for empty token
	if a.token == "" {
		return nil, nil
	}

	// put code to header
	h := map[string]string{"X-Auth-Sms-Token": a.token, "X-Auth-Sms-Code": a.code}

	// clear data
	a.token = ""
	a.code = ""

	// done
	return h, nil
}

func NewAuthSms() *AuthSms {
	c := &AuthSms{}
	return c
}
