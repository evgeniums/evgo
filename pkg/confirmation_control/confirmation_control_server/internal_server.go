package confirmation_control_server

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/api/pool_microservice/pool_microservice_server"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/confirmation_control/confirmation_control_api/confirmation_api_service"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type InternalServerConfig struct {
	TOKEN_TTL int `default:"180" validate:"gte=1" vmessage:"Token TTL must be positive"`
}

const InternalServerType = "confirmation_control_internal"

type InternalServer struct {
	InternalServerConfig

	basePublicUrl string
	*pool_microservice_server.PoolMicroserviceServer
}

func NewInternalServer() *InternalServer {
	s := &InternalServer{}
	return s
}

func (s *InternalServer) Config() interface{} {
	return &s.InternalServerConfig
}

func (s *InternalServer) Init(app app_with_multitenancy.AppWithMultitenancy, sctx context.Context, basePublicUrl string, configPath ...string) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("InternalServer.Init")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	s.basePublicUrl = basePublicUrl
	path := utils.OptionalArg("internal_server", configPath...)

	err = object_config.LoadLogValidate(app.Cfg(), app.Logger(), app.Validator(), s, path)
	if err != nil {
		c.SetMessage("failed to init internal server of confirmation control")
		return err
	}

	// init microservice server for internal requests
	s.PoolMicroserviceServer = pool_microservice_server.New(InternalServerType)
	err = s.PoolMicroserviceServer.Init(app, path)
	if err != nil {
		c.SetMessage("to init microservice server for internal server")
		return err
	}
	if s.basePublicUrl == "" {
		err = errors.New("public URL must be not empty")
		return err
	}

	// create and add service
	service := confirmation_api_service.NewConfirmationInternalService(s.basePublicUrl, s.TOKEN_TTL)
	api_server.AddServiceToServer(s.PoolMicroserviceServer.ApiServer(), service)

	// done
	return nil
}
