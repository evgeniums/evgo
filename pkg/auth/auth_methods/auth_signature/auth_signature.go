package auth_signature

import (
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/signature"
	"github.com/evgeniums/evgo/pkg/signature/user_pubkey"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

const SignatureProtocol = "signature"
const SignatureParameter = "signature"

type AuthSignatureConfig struct {
}

type AuthSignature struct {
	auth.AuthHandlerBase
	AuthSignatureConfig
	signatureManager signature.SignatureManager
}

func (a *AuthSignature) Config() interface{} {
	return &a.AuthSignatureConfig
}

func New(manager signature.SignatureManager) *AuthSignature {
	a := &AuthSignature{}
	a.signatureManager = manager
	return a
}

func (a *AuthSignature) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	a.AuthHandlerBase.Init(SignatureProtocol)

	err := object_config.LoadLogValidate(cfg, log, vld, a, "auth.methods.signature", configPath...)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of Signature auth handler", err)
	}
	return nil
}

func (a *AuthSignature) ErrorDescriptions() map[string]string {
	h := signature.ErrorDescriptions
	utils.AppendMap(h, user_pubkey.ErrorDescriptions)
	return h
}

func (a *AuthSignature) ErrorProtocolCodes() map[string]int {
	h := signature.ErrorHttpCodes
	utils.AppendMap(h, user_pubkey.ErrorHttpCodes)
	return h
}

// Check signature in request.
// Call this handler after discovering user (ctx.AuthUser() must be not nil).
// Public key of user must be set for the user.
// signature is calculated as sig(sha256(RequestContent,RequestMethod,RequestPath))
func (a *AuthSignature) Handle(ctx auth.AuthContext) (bool, error) {

	// setup
	c := ctx.TraceInMethod("AuthSignature.Handle")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// get token from request
	requestSignature := ctx.GetAuthParameter(a.Protocol(), SignatureParameter)
	if requestSignature == "" {
		c.Logger().Info("skip signature auth")
		return false, nil
	}

	// verify signature
	err = a.signatureManager.Verify(ctx, requestSignature, ctx.GetRequestContent(), ctx.GetRequestMethod(), ctx.GetRequestPath())
	if err != nil {
		return true, err
	}

	// done
	return true, nil
}

func (a *AuthSignature) SetAuthManager(manager auth.AuthManager) {
	manager.Schemas().AddHandler(a)
}
