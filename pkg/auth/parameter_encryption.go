package auth

import (
	"context"
	"errors"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/message/message_json"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type AuthParameterEncryption interface {
	Encrypt(sctx context.Context, obj interface{}) (string, error)
	SetAuthParameter(sctx context.Context, authMethodProtocol string, name string, obj interface{}, directKeyName ...bool) error
	GetAuthParameter(sctx context.Context, authMethodProtocol string, name string, obj interface{}, tag string, directKeyName ...bool) (bool, error)

	DecodeAuthParameter(sctx context.Context, name string, value string, obj interface{}, tag ...string) (bool, error)
	EncodeAuthParameter(sctx context.Context, name string, obj interface{}) (string, error)

	CurrentTag() string
}

// TODO Keep outdated secrets with tags

type AuthParameterEncryptionBaseConfig struct {
	SECRET            string `validate:"required" mask:"true"`
	PBKDF2_ITERATIONS uint   `default:"256"`
	SALT_SIZE         int    `default:"8" validate:"lte=32,gte=4"`
	TAG               string
}

type AuthParameterEncryptionBase struct {
	AuthParameterEncryptionBaseConfig
	Serializer   message.Serializer
	StringCoding utils.StringCoding
}

func (a *AuthParameterEncryptionBase) Config() interface{} {
	return &a.AuthParameterEncryptionBaseConfig
}

func (a *AuthParameterEncryptionBase) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {
	a.Serializer = &message_json.JsonSerializer{}
	a.StringCoding = &utils.Base64StringCoding{}

	err := object_config.LoadLogValidate(cfg, log, vld, a, "auth.params_encryption", configPath...)
	if err != nil {
		return log.PushFatalStack("failed to load configuration of auth parameters encryption", err)
	}
	return nil
}

func (a *AuthParameterEncryptionBase) createCipher(salt []byte, tag ...string) (*crypt_utils.AEAD, error) {
	pbkdfCfg := crypt_utils.DefaultPbkdfConfig()
	pbkdfCfg.Iter = int(a.PBKDF2_ITERATIONS)
	aeadCfg := crypt_utils.DefaultAEADConfig(pbkdfCfg)

	secret := a.SECRET
	// TODO find outdated secret by tag

	cipher, err := crypt_utils.NewAEAD(secret, salt, aeadCfg)
	return cipher, err
}

func (a *AuthParameterEncryptionBase) EncodeAuthParameter(sctx context.Context, name string, obj interface{}) (string, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AuthParameterEncryptionBase.EncodeAuthParameter", logger.Fields{"name": name})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// encode data to string
	data, err := a.Encrypt(sctx, obj)
	if err != nil {
		c.SetMessage("failed to encrypt")
		return "", err
	}

	// done
	return data, nil
}

func (a *AuthParameterEncryptionBase) SetAuthParameter(sctx context.Context, authMethodProtocol string, name string, obj interface{}, directKeyName ...bool) error {

	// encode
	ctx := op_context.OpContext[AuthContext](sctx)
	data, err := a.EncodeAuthParameter(sctx, name, obj)
	if err != nil {
		return err
	}

	// write result to  auth parameter
	ctx.SetAuthParameter(authMethodProtocol, name, data, directKeyName...)

	// done
	return nil
}

func EncodeAndSetBearer(sctx context.Context, enc AuthParameterEncryption, obj interface{}) error {
	value, err := enc.EncodeAuthParameter(sctx, AuthorizationBearer, obj)
	if err != nil {
		return err
	}
	ctx := op_context.OpContext[AuthContext](sctx)
	SetAuthBearer(ctx, value)
	return nil
}

func (a *AuthParameterEncryptionBase) GetAuthParameter(sctx context.Context, authMethodProtocol string, name string, obj interface{}, tag string, directKeyName ...bool) (bool, error) {

	// read auth parameter
	ctx := op_context.OpContext[AuthContext](sctx)
	data := ctx.GetAuthParameter(authMethodProtocol, name, directKeyName...)
	if data == "" {
		return false, nil
	}
	return a.DecodeAuthParameter(sctx, name, data, obj, tag)
}

func (a *AuthParameterEncryptionBase) DecodeAuthParameter(sctx context.Context, name string, value string, obj interface{}, tag ...string) (bool, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AuthParameterEncryptionBase.DecodeAuthParameter", logger.Fields{"name": name})
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// decode data
	ciphertext, err := a.StringCoding.Decode(value)
	if err != nil {
		c.SetMessage("failed to decode data")
		return true, err
	}

	// split data to salt and ciphertext
	if len(ciphertext) < a.SALT_SIZE {
		err := errors.New("ciphertext too short for salt")
		return true, err
	}
	salt := ciphertext[len(ciphertext)-a.SALT_SIZE:]
	ciphertext = ciphertext[:len(ciphertext)-len(salt)]

	// create cipher
	cipher, err := a.createCipher(salt, tag...)
	if err != nil {
		c.SetMessage("failed to create AEAD cipher")
		return true, err
	}

	// decrypt data
	plaintext, err := cipher.Decrypt(ciphertext)
	if err != nil {
		c.SetMessage("failed to decrypt ciphertext")
		return true, err
	}

	// parse message
	err = a.Serializer.ParseMessage(plaintext, obj)
	if err != nil {
		c.SetMessage("failed to parse plaintext")
		return true, err
	}

	// done
	return true, nil
}

func GetAndDecodeBearer(sctx context.Context, enc AuthParameterEncryption, obj interface{}) (bool, error) {
	ctx := op_context.OpContext[AuthContext](sctx)
	value := GetAuthBearer(ctx)
	if value == "" {
		return false, nil
	}
	return enc.DecodeAuthParameter(sctx, AuthorizationBearer, value, obj)
}

func (a *AuthParameterEncryptionBase) Encrypt(sctx context.Context, obj interface{}) (string, error) {

	// setup
	ctx := op_context.OpContext[op_context.Context](sctx)
	c := ctx.TraceInMethod("AuthParameterEncryptionBase.Encrypt")
	var err error
	onExit := func() {
		if err != nil {
			c.SetError(err)
		}
		ctx.TraceOutMethod()
	}
	defer onExit()

	// serialize object to plaintext
	plaintext, err := a.Serializer.SerializeMessage(obj)
	if err != nil {
		c.SetMessage("failed to serialize object")
		return "", err
	}

	// generate salt
	salt, err := crypt_utils.GenerateCryptoRand(a.SALT_SIZE)
	if err != nil {
		c.SetMessage("failed to generate salt")
		return "", err
	}

	// create cipher
	cipher, err := a.createCipher(salt)
	if err != nil {
		c.SetMessage("failed to create AEAD cipher")
		return "", err
	}

	// encrypt data
	ciphertext, err := cipher.Encrypt(plaintext)
	if err != nil {
		c.SetMessage("failed to encrypt data")
		return "", err
	}

	// append salt to ciphertext
	ciphertext = append(ciphertext, salt...)

	// encode data to string
	data := a.StringCoding.Encode(ciphertext)

	// done
	return data, nil
}

func (a *AuthParameterEncryptionBase) CurrentTag() string {
	return a.TAG
}
