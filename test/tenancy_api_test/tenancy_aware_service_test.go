package tenancy_api_test

import (
	"context"
	"testing"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_client"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_token"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/pubsub/pubsub_providers/pubsub_factory"
	"github.com/evgeniums/evgo/pkg/test_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SampleService struct {
	api_server.ServiceBase
	SampleResource api.Resource
}

func NewSampleService() *SampleService {

	s := &SampleService{}

	s.Init("samples", "tenancy_test", true)
	s.SampleResource = api.NewResource("sample")
	s.AddChild(s.SampleResource)
	s.SampleResource.AddOperation(List(s))

	return s
}

type SampleEndpoint struct {
	service *SampleService
	api_server.EndpointBase
}

func (e *SampleEndpoint) Construct(service *SampleService, op api.Operation) {
	e.service = service
	e.EndpointBase.Construct(op)
}

type ListEndpoint struct {
	SampleEndpoint
}

type ListResponse = api.ResponseList[*InTenancySample]

func (e *ListEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	var err error
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("samples.List")
	defer request.TraceOutMethod()

	// get
	resp := &ListResponse{}
	resp.Count, err = request.Db().FindWithFilter(sctx, nil, &resp.Items)
	if err != nil {
		return sctx, c.SetError(err)
	}
	// set response message
	request.Response().SetMessage(resp)

	// done
	return sctx, nil
}

func List(s *SampleService) *ListEndpoint {
	e := &ListEndpoint{}
	e.Construct(s, api.List("list_samples"))
	return e
}

type SampleClient struct {
	api_client.ServiceClient
	SampleResource api.Resource
	list           api.Operation
}

func NewSampleClient(client api_client.Client) *SampleClient {

	c := &SampleClient{}

	c.Init(client, "samples")
	c.SampleResource = api.NewResource("sample")
	c.AddChild(c.SampleResource)
	c.list = api.List("list_samples")

	c.SampleResource.AddOperations(
		c.list,
	)

	return c
}

type ClientList struct {
	result *ListResponse
}

func (l *ClientList) Exec(client api_client.Client, sctx context.Context, operation api.Operation) error {

	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("ClientList")
	defer ctx.TraceOutMethod()

	err := client.Exec(sctx, operation, nil, l.result)
	c.SetError(err)
	return err
}

func (s *SampleClient) List(sctx context.Context) ([]*InTenancySample, error) {

	// setup
	var err error
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("SampleClient.List")
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// prepare and exec handler
	handler := &ClientList{
		result: &ListResponse{},
	}
	err = handler.Exec(s.Client(), sctx, s.list)
	if err != nil {
		c.SetMessage("failed to exec operation")
		return nil, err
	}

	// done
	return handler.result.Items, nil
}

func TestTenancyAwareService(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add first tenancy to the same pool as single pool app, add via mutipool app
	tenancyData1 := &multitenancy.TenancyData{}
	tenancyData1.POOL_ID = "pool2"
	tenancyData1.ROLE = "dev"
	tenancyData1.DESCRIPTION = "tenancy for development"
	tenancyData1.CUSTOMER_ID = "customer1"
	addedTenancy1, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData1)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy1)
	loadedTenancy1, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)

	// add second tenancy to the same pool as single pool app, add via mutipool app
	tenancyData2 := tenancyData1
	tenancyData2.CUSTOMER_ID = "customer2"
	addedTenancy2, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData2)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy2)
	err = multiPoolCtx.RemoteTenancyController.Deactivate(op_context.MakeOpContext(multiPoolCtx.ClientOp), addedTenancy2.GetID())
	require.NoError(t, err)
	loadedTenancy2, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy2.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy2)
	assert.False(t, loadedTenancy2.IsBlockedPath())

	// add third tenancy to the same pool as single pool app, add via mutipool app then block path
	tenancyData3 := tenancyData1
	tenancyData3.ROLE = "blocked_path_role"
	addedTenancy3, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData3)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy3)
	err = multiPoolCtx.RemoteTenancyController.SetPathBlocked(op_context.MakeOpContext(multiPoolCtx.ClientOp), addedTenancy3.GetID(), true, multitenancy.TenancyBlockPathModeDefault)
	require.NoError(t, err)
	loadedTenancy3, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy3.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy3)
	assert.True(t, loadedTenancy3.IsBlockedPath())

	// add document to tenancy database
	sample1 := &InTenancySample{Field1: "hello world", Field2: 10}
	err = loadedTenancy1.Db().Create(op_context.MakeOpContext(multiPoolCtx.AdminOp), sample1)
	require.NoError(t, err)

	// add sample service
	sampleService := NewSampleService()
	api_server.AddServiceToServer(multiPoolCtx.Server.ApiServer(), sampleService)

	// create sample client
	sampleClient := NewSampleClient(multiPoolCtx.RestApiClient)
	tenancyResource := api.NamedResource("tenancy")
	tenancyResource.AddChild(sampleClient)

	// invoke operation on tenancy 1
	tenancyResource.SetId(loadedTenancy1.Path())
	samples, err := sampleClient.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	require.NoError(t, err)
	require.NotNil(t, samples)
	require.Equal(t, 1, len(samples))
	assert.Equal(t, sample1, samples[0])

	// invoke operation on tenancy 2 that is not active
	tenancyResource.SetId(loadedTenancy2.Path())
	samples, err = sampleClient.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeNotFound)
	assert.Equal(t, 0, len(samples))

	// invoke operation on tenancy 3 that is blocked
	tenancyResource.SetId(loadedTenancy3.Path())
	samples, err = sampleClient.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	require.NoError(t, err)
	require.NotNil(t, samples)
	assert.Equal(t, 0, len(samples))

	// try to invoke operation on unknown tenancy
	tenancyResource.SetId("unknowntenancypath")
	_, err = sampleClient.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeNotFound)

	// init service for singlepool app
	singlePoolSampleService := NewSampleService()
	api_server.AddServiceToServer(singlePoolCtx.Server.ApiServer(), singlePoolSampleService)
	singlePoolSampleClient := NewSampleClient(singlePoolCtx.RestApiClient)
	singlePoolTenancyResource := api.NamedResource("tenancy")
	singlePoolTenancyResource.AddChild(singlePoolSampleClient)

	// try to invoke operation in single pool app where auth is from tenancies database
	singlePoolTenancyResource.SetId(loadedTenancy1.Path())
	_, err = singlePoolSampleClient.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	test_utils.CheckGenericError(t, err, auth_token.ErrorCodeSessionExpired)

	// close apps
	multiPoolCtx.Close()
	singlePoolCtx.Close()

	// check disallowed blocked tenancy
	multiPoolCtx = initContext(t, false, "tenancy_notactive")
	sampleService1 := NewSampleService()
	api_server.AddServiceToServer(multiPoolCtx.Server.ApiServer(), sampleService1)
	sampleClient1 := NewSampleClient(multiPoolCtx.RestApiClient)
	tenancyResource1 := api.NamedResource("tenancy")
	tenancyResource1.AddChild(sampleClient1)
	tenancyResource1.SetId(loadedTenancy1.Path())
	samples, err = sampleClient1.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	require.NoError(t, err)
	require.NotNil(t, samples)
	require.Equal(t, 1, len(samples))
	assert.Equal(t, sample1, samples[0])
	tenancyResource1.SetId(loadedTenancy3.Path())
	_, err = sampleClient1.List(op_context.MakeOpContext(multiPoolCtx.ClientOp))
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeNotFound)

	// close apps
	multiPoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}

func TestServerIpList(t *testing.T) {

	// prepare app with multiple pools and single pool
	multiPoolCtx, singlePoolCtx := PrepareAppWithTenancies(t)

	// add first tenancy to the same pool as single pool app, add via mutipool app
	tenancyData1 := &multitenancy.TenancyData{}
	tenancyData1.POOL_ID = "pool2"
	tenancyData1.ROLE = "dev"
	tenancyData1.DESCRIPTION = "tenancy for development"
	tenancyData1.CUSTOMER_ID = "customer1"
	addedTenancy1, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData1)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy1)
	err = multiPoolCtx.RemoteTenancyController.Activate(op_context.MakeOpContext(multiPoolCtx.ClientOp), addedTenancy1.GetID())
	require.NoError(t, err)
	loadedTenancy1, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy1.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)

	// add IP address to tenancy 1
	err = multiPoolCtx.RemoteTenancyController.AddIpAddress(op_context.MakeOpContext(multiPoolCtx.ClientOp), addedTenancy1.GetID(), "127.0.0.1", "internal")
	require.NoError(t, err)

	// add second tenancy to the same pool as single pool app, add via mutipool app
	tenancyData2 := tenancyData1
	tenancyData2.CUSTOMER_ID = "customer2"
	addedTenancy2, err := multiPoolCtx.RemoteTenancyController.Add(op_context.MakeOpContext(multiPoolCtx.ClientOp), tenancyData2)
	require.NoError(t, err)
	require.NotNil(t, addedTenancy2)
	loadedTenancy2, err := multiPoolCtx.AppWithTenancy.Multitenancy().Tenancy(addedTenancy2.GetID())
	require.NoError(t, err)
	require.NotNil(t, loadedTenancy1)

	// check IP filtering
	multiPoolCtx1 := initContext(t, false, "filter_ip")
	sampleService2 := NewSampleService()
	api_server.AddServiceToServer(multiPoolCtx1.Server.ApiServer(), sampleService2)
	sampleClient2 := NewSampleClient(multiPoolCtx1.RestApiClient)
	tenancyResource2 := api.NamedResource("tenancy")
	tenancyResource2.AddChild(sampleClient2)
	tenancyResource2.SetId(loadedTenancy1.Path())

	// good
	_, err = sampleClient2.List(op_context.MakeOpContext(multiPoolCtx1.ClientOp))
	assert.NoError(t, err)

	// filter because no addresses
	tenancyResource2.SetId(loadedTenancy2.Path())
	_, err = sampleClient2.List(op_context.MakeOpContext(multiPoolCtx1.ClientOp))
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeForbidden)

	// filter because good address but with other tag
	err = multiPoolCtx1.RemoteTenancyController.AddIpAddress(op_context.MakeOpContext(multiPoolCtx1.ClientOp), addedTenancy2.GetID(), "127.0.0.1", "external")
	require.NoError(t, err)
	_, err = sampleClient2.List(op_context.MakeOpContext(multiPoolCtx1.ClientOp))
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeForbidden)

	// filter because address with self tag not matching localhost
	err = multiPoolCtx1.RemoteTenancyController.AddIpAddress(op_context.MakeOpContext(multiPoolCtx1.ClientOp), addedTenancy2.GetID(), "192.168.100.1", "internal")
	require.NoError(t, err)
	_, err = sampleClient2.List(op_context.MakeOpContext(multiPoolCtx1.ClientOp))
	test_utils.CheckGenericError(t, err, generic_error.ErrorCodeForbidden)

	// good
	err = multiPoolCtx1.RemoteTenancyController.AddIpAddress(op_context.MakeOpContext(multiPoolCtx1.ClientOp), addedTenancy2.GetID(), "127.0.0.1", "internal")
	require.NoError(t, err)
	_, err = sampleClient2.List(op_context.MakeOpContext(multiPoolCtx1.ClientOp))
	assert.NoError(t, err)

	// close apps
	multiPoolCtx1.Close()
	multiPoolCtx.Close()
	singlePoolCtx.Close()
	pubsub_factory.ResetSingletonInmemPubsub()
}
