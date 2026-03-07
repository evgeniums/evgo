package auth_factory

import (
	"fmt"

	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_hmac"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_signature"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_sms"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_token"
	"github.com/evgeniums/go-utils/pkg/auth/auth_session"
	"github.com/evgeniums/go-utils/pkg/signature"
	"github.com/evgeniums/go-utils/pkg/sms"
)

type DefaultAuthFactory struct {
	Users            auth_session.WithUserSessionManager
	SmsManager       sms.SmsManager
	SignatureManager signature.SignatureManager
}

func (f *DefaultAuthFactory) Create(protocol string) (auth.AuthHandler, error) {

	switch protocol {
	case LoginPhashTokenProtocol:
		return NewLoginphashToken(f.Users), nil
	case LoginPhashSmsTokenProtocol:
		return NewLoginphashSmsToken(f.Users, f.SmsManager), nil
	case auth_login_phash.LoginProtocol:
		return auth_login_phash.New(f.Users), nil
	case auth_token.TokenProtocol:
		return auth_token.New(f.Users), nil
	case auth_token.GenTokenProtocol:
		return auth_token.NewSchema(f.Users), nil
	case auth_hmac.HmacProtocol:
		return &auth_hmac.AuthHmac{}, nil
	case auth_sms.SmsProtocol:
		return auth_sms.New(f.SmsManager), nil
	case auth_signature.SignatureProtocol:
		return auth_signature.New(f.SignatureManager), nil
	case auth.NoAuthProtocol:
		return &auth.NoAuthMethod{}, nil
	}

	return nil, fmt.Errorf("unknown auth handler %s", protocol)
}
