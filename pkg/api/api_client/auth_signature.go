package api_client

import (
	"encoding/json"

	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_signature"
	"github.com/evgeniums/go-utils/pkg/config"
	"github.com/evgeniums/go-utils/pkg/config/object_config"
	"github.com/evgeniums/go-utils/pkg/crypt_utils"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/utils"
	"github.com/evgeniums/go-utils/pkg/validator"
)

type AuthSignature struct {
	Signer          crypt_utils.ESigner
	EndpoindsConfig auth.EndpointsAuthConfig
}

func (a *AuthSignature) HandleResponse(resp Response) {
}

func (a *AuthSignature) MakeHeaders(ctx op_context.Context, operation api.Operation, cmd interface{}) (map[string]string, error) {

	// setup
	c := ctx.TraceInMethod("AuthSignature.MakeHeaders")
	defer ctx.TraceOutMethod()

	// find auth schema for operation
	path := operation.Resource().ServicePathPrototype()
	schema, found := a.EndpoindsConfig.Schema(path, operation.AccessType())
	if !found || schema != auth_signature.SignatureProtocol {
		return nil, nil
	}

	// serialize command
	content, err := json.Marshal(cmd)
	if err != nil {
		c.SetMessage("failed to marshal command")
		return nil, c.SetError(err)
	}

	// sign request
	sig, err := a.Signer.SignB64(content, access_control.Access2HttpMethod(operation.AccessType()), path)
	if err != nil {
		c.SetMessage("failed to sign request")
		return nil, c.SetError(err)
	}

	// put signature to header
	h := map[string]string{"x-auth-signature": sig}

	// done
	return h, nil
}

type AuthSignatureBaseConfig struct {
	PRIVATE_KEY_FILE     string `validate:"required"`
	PRIVATE_KEY_PASSWORD string `mask:"true"`
}

type AuthSignatureBase struct {
	AuthSignatureBaseConfig
	AuthSignature

	rsaSigner       *crypt_utils.RsaSigner
	endpointsConfig *auth.EndpointsAuthConfigBase
}

func (a *AuthSignatureBase) Config() interface{} {
	return &a.AuthSignatureBaseConfig
}

func (a *AuthSignatureBase) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	path := utils.OptionalArg("auth_signature", configPath...)

	// load config
	err := object_config.LoadLogValidate(cfg, log, vld, a, path)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of client auth signature", err)
	}

	// load key
	if a.rsaSigner != nil {
		err = a.rsaSigner.LoadKeyFromFile(a.PRIVATE_KEY_FILE, a.PRIVATE_KEY_PASSWORD)
		if err != nil {
			return log.PushFatalStack("failed to load private key for RSA signer of client auth signature", err)
		}
	}

	// load endpoints configuration
	endpointesPath := object_config.Key(path, "endpoints")
	err = a.endpointsConfig.Init(cfg, log, vld, endpointesPath)
	if err != nil {
		return log.PushFatalStack("failed to load endpoints configuration of client auth signature", err)
	}

	// done
	return nil
}

func NewAuthSignature() *AuthSignatureBase {
	c := &AuthSignatureBase{}
	c.Construct()
	return c
}

func (c *AuthSignatureBase) Construct() {
	c.rsaSigner = crypt_utils.NewRsaSigner()
	c.Signer = c.rsaSigner
	c.endpointsConfig = auth.NewEndpointsAuthConfigBase()
	c.EndpoindsConfig = c.endpointsConfig
}
