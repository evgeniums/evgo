package confirmation_control_server

import (
	"context"

	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/api/noauth_server"
	"github.com/evgeniums/evgo/pkg/api/pool_microservice/pool_misrocervice_client"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/confirmation_control/confirmation_control_api/confirmation_api_client"
	"github.com/evgeniums/evgo/pkg/confirmation_control/confirmation_control_api/confirmation_api_service"
	"github.com/evgeniums/evgo/pkg/multitenancy/app_with_multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/sms"
	"github.com/evgeniums/evgo/pkg/utils"
)

const ExternalServerType string = "confirmation_control_external"

type ExternalServerConfig struct {
	EXPLICIT_CODE_CHECK bool
	SMS_DB_SERVICE_ROLE string
}

type ExternalServer struct {
	ExternalServerConfig

	auth         auth.Auth
	server       api_server.Server
	smsManager   sms.SmsManager
	smsProviders sms.ProviderFactory

	callbackTransport *pool_misrocervice_client.PoolMicroserviceClient
}

type ExternalServerCfg struct {
	SmsManager   sms.SmsManager
	SmsProviders sms.ProviderFactory
}

func (s *ExternalServer) Construct(config ...ExternalServerCfg) {
	if len(config) != 0 {
		cfg := config[0]
		s.smsManager = cfg.SmsManager
		s.smsProviders = cfg.SmsProviders
	}
}

func NewExternalServer(config ...ExternalServerCfg) *ExternalServer {
	s := &ExternalServer{}
	s.Construct(config...)
	return s
}

func (s *ExternalServer) Config() interface{} {
	return &s.ExternalServerConfig
}

func (s *ExternalServer) Init(app app_with_multitenancy.AppWithMultitenancy, sctx context.Context, configPath ...string) error {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("ExternalServer.Init")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()
	path := utils.OptionalArg("external_server", configPath...)

	// load config
	err = object_config.LoadLogValidate(app.Cfg(), app.Logger(), app.Validator(), s, path)
	if err != nil {
		c.SetMessage("failed to init external server of confirmation control")
		return err
	}

	// init SMS manager
	if s.smsManager == nil && s.smsProviders != nil && app.Cfg().IsSet("sms") {
		smsManager := sms.NewSmsManager()
		err := smsManager.Init(app.Cfg(), app.Logger(), app.Validator(), s.smsProviders, "sms")
		if err != nil {
			c.SetMessage("failed to init SMS manager")
			return err
		}
		if s.SMS_DB_SERVICE_ROLE != "" {
			selfPool, err := app.Pools().SelfPool()
			if err != nil {
				c.SetMessage("self pool must be specified")
				return err
			}
			err = smsManager.InitDbService(sctx, selfPool, s.SMS_DB_SERVICE_ROLE)
			if err != nil {
				c.SetMessage("failed to init database service for SMS manager")
				return err
			}
		}
		s.smsManager = smsManager
	}

	// init auth controller
	if s.auth == nil {
		auth := auth.NewAuth()
		authPath := object_config.Key(path, "auth")
		err := auth.Init(app.Cfg(), app.Logger(), app.Validator(), &AuthFactory{SmsManager: s.smsManager}, authPath)
		if err != nil {
			c.SetMessage("failed to init auth manager")
			return err
		}
		s.auth = auth
	}

	serverCfg := noauth_server.Config{DefaultPoolServiceType: ExternalServerType, Auth: s.auth}
	server := noauth_server.New(serverCfg)
	err = server.Init(app, path)
	if err != nil {
		c.SetMessage("failed to init noauth server")
		return err
	}
	s.server = server.ApiServer()

	// create callback client
	callbackTransportPath := object_config.Key(path, "callback_client")
	s.callbackTransport = pool_misrocervice_client.NewPoolMicroserviceClient("confirmation_callback")
	err = s.callbackTransport.Init(app, callbackTransportPath)
	if err != nil {
		c.SetMessage("failed to init callback client")
		return err
	}
	s.callbackTransport.SetPropagateAuthUser(false)
	callbackClient := confirmation_api_client.NewConfirmationCallbackClient(s.callbackTransport)

	// add services
	externalService := confirmation_api_service.NewConfirmationExternalService(callbackClient, s.EXPLICIT_CODE_CHECK)
	api_server.AddServiceToServer(s.server, externalService)

	// done
	return nil
}

func (s *ExternalServer) ApiServer() api_server.Server {
	return s.server
}

func (s *ExternalServer) CallbackMicroserviceClient() *pool_misrocervice_client.PoolMicroserviceClient {
	return s.callbackTransport
}

func (s *ExternalServer) BaseUrl() string {

	poolService := s.server.ConfigPoolService()

	return poolService.PublicUrl()
}
