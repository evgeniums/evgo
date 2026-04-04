package filedata_service

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/op_context"
)

const ServiceName = "filedata"
const FetchName = "fetch"
const UploadName = "upload"

type FileDataService struct {
	api_server.ServiceBase
	controller FileDataController

	fetch  *FetchEndpoint
	upload *UploadEndpoint
}

func NewFileDataService(controller FileDataController, multitenancy ...bool) *FileDataService {
	s := &FileDataService{controller: controller}

	s.Init(ServiceName, api.PackageName, multitenancy...)

	var fetchResourcesChain []string
	var uploadResourcesChain []string
	if controller.UrlManager().IsTopicEnabled() {
		fetchResourcesChain = []string{controller.UrlManager().TopicUrlParameter(), controller.UrlManager().IdUrlPathParameter()}
		uploadResourcesChain = []string{controller.UrlManager().TopicUrlParameter(), controller.UrlManager().IdUrlPathParameter(), controller.UrlManager().PartUrlPathParameter()}
	} else {
		fetchResourcesChain = []string{controller.UrlManager().IdUrlPathParameter()}
		uploadResourcesChain = []string{controller.UrlManager().IdUrlPathParameter(), controller.UrlManager().PartUrlPathParameter()}
	}

	s.fetch = NewFetchEndpoint(s)
	fetchResource := api.NewIdResourcesChain(s.fetch.Name(), fetchResourcesChain...)
	fetchResource.AddOperation(s.fetch)

	s.upload = NewUploadEndpoint(s)
	uploadResource := api.NewIdResourcesChain(s.upload.Name(), uploadResourcesChain...)
	uploadResource.AddOperation(s.upload)

	s.AddChildren(
		fetchResource,
		uploadResource,
	)
	return s
}

type Endpoint struct {
	api_server.EndpointBase
	service *FileDataService
}

func (ep *Endpoint) Construct(service *FileDataService, op api.Operation) {
	ep.EndpointBase.Construct(op)
	ep.service = service
}

var Fetch = func() api.Operation { return api.Get(FetchName) }

type FetchEndpoint struct {
	Endpoint
}

func (e *FetchEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("FetchEndpoint")
	defer request.TraceOutMethod()

	// invoke
	err := e.service.controller.Fetch(sctx)
	if err != nil {
		c.SetMessage("operation failed")
		return sctx, err
	}

	// done
	return sctx, nil
}

func NewFetchEndpoint(service *FileDataService) *FetchEndpoint {
	ep := &FetchEndpoint{}
	ep.Construct(service, Fetch())
	return ep
}

var Upload = func() api.Operation { return api.Get(UploadName) }

type UploadEndpoint struct {
	Endpoint
}

func (e *UploadEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {

	// setup
	request := op_context.OpContext[api_server.Request](sctx)
	c := request.TraceInMethod("UploadEndpoint")
	defer request.TraceOutMethod()

	// invoke
	err := e.service.controller.UploadPart(sctx)
	if err != nil {
		c.SetMessage("operation failed")
		return sctx, err
	}

	// done
	return sctx, nil
}

func (e *UploadEndpoint) IsFileUpload() bool {
	return true
}

func NewUploadEndpoint(service *FileDataService) *UploadEndpoint {
	ep := &UploadEndpoint{}
	ep.Construct(service, Fetch())
	return ep
}
