package confirmation_control_server

import (
	"fmt"

	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/auth/auth_methods/auth_sms"
	"github.com/evgeniums/evgo/pkg/confirmation_control/sms_auth_confirmation"
	"github.com/evgeniums/evgo/pkg/sms"
)

type AuthFactory struct {
	SmsManager sms.SmsManager
}

func (f *AuthFactory) Create(protocol string) (auth.AuthHandler, error) {

	switch protocol {
	case auth_sms.SmsProtocol:
		return auth_sms.New(f.SmsManager), nil
	case auth.NoAuthProtocol:
		return &auth.NoAuthMethod{}, nil
	case sms_auth_confirmation.CachedPhoneAuthProtocol:
		return &sms_auth_confirmation.CachedPhoneAuthMethod{}, nil
	}

	return nil, fmt.Errorf("unknown auth handler %s", protocol)
}
