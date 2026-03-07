package tenancy_client

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/multitenancy/tenancy_api"
	"github.com/evgeniums/evgo/pkg/op_context"
)

func (t *TenancyClient) DeleteIpAddress(ctx op_context.Context, id string, ipAddress string, tag string, idIsDisplay ...bool) error {

	// setup
	var err error
	c := ctx.TraceInMethod("TenancyClient.AddIpAddress")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// adjust ID
	tenancyId, _, err := multitenancy.TenancyId(t, ctx, id, idIsDisplay...)
	if err != nil {
		c.SetMessage("failed to get tenancy ID")
		return err
	}

	// prepare and exec handler
	cmd := &tenancy_api.IpAddressCmd{
		Ip:  ipAddress,
		Tag: tag,
	}
	handler := api_client.NewHandlerRequest(cmd)
	op := api.OperationAsResource(t.TenancyResource, tenancy_api.IpAddressResource, tenancyId, tenancy_api.DeleteIpAddress())
	err = handler.Exec(t.Client(), ctx, op)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return err
	}

	// done
	return nil
}
