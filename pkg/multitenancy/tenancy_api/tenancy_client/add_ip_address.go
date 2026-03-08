package tenancy_client

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) AddIpAddress(sctx context.Context, id string, ipAddress string, tag string, idIsDisplay ...bool) error {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("TenancyClient.AddIpAddress")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust ID
	tenancyId, _, err := multitenancy.TenancyId(t, sctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to get tenancy ID")
		return err
	}

	// prepare and exec handler
	request := &tenancy_api.IpAddressCmd{
		Ip:  ipAddress,
		Tag: tag,
	}
	handler := api_client.NewHandlerRequest(request)
	op := api.OperationAsResource(t.TenancyResource, tenancy_api.IpAddressResource, tenancyId, tenancy_api.AddIpAddress())
	err = handler.Exec(t.Client(), sctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
