package grpc_api

import (
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
)

const PackageName = "grpc_api"

type GrpcTestService struct {
	api_server.ServiceBase
}

func NewGrpcTestService(multitenancy ...bool) *GrpcTestService {
	s := &GrpcTestService{}

	s.InitExplicit("grpc-test", "GrpcTest", PackageName, multitenancy...)

	basic := api.NewResource("basic")
	basic.AddOperation(NewBasicEndpoint())
	s.AddChild(basic)

	repeated := api.NewResource("repeated")
	repeated.AddOperation(NewRepeatedEndpoint())
	s.AddChild(repeated)

	embedded := api.NewResource("embedded")
	embedded.AddOperation(NewEmbeddedEndpoint())
	s.AddChild(embedded)

	echoToken := api.NewResource("echo-token")
	echoToken.AddOperation(NewEchoTokenEndpoint())
	s.AddChild(echoToken)

	return s
}
