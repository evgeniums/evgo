package filestorage

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"time"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/utils"
)

type SignedUrlConfig struct {
	SECRET          string `validate:"required" vmessage:"secret must be defined for signed URLs"`
	EXPIRATION      uint32 `default:"3600"`
	EXPIRY_PARAM    string `default:"e"`
	SIGNATURE_PARAM string `default:"s"`
}

type SignedUrlHandlerBase struct {
	SignedUrlConfig
	hmacBuilder crypt_utils.HmacBuilder
}

func NewSignedUrl(hmacBuilder ...crypt_utils.HmacBuilder) *SignedUrlHandlerBase {
	s := &SignedUrlHandlerBase{}

	if len(hmacBuilder) != 0 && hmacBuilder[0] != nil {
		s.hmacBuilder = hmacBuilder[0]
	} else {
		s.hmacBuilder = func(secret string) *crypt_utils.Hmac { return crypt_utils.NewHmac(secret) }
	}

	return s
}

func (s *SignedUrlHandlerBase) Config() any {
	return &s.SignedUrlConfig
}

func (s *SignedUrlHandlerBase) Init(app app_context.Context, parentConfigPath string, configPath ...string) error {

	path := utils.OptionalString(object_config.Key(parentConfigPath, "signed_url"), configPath...)
	err := object_config.LoadLogValidateApp(app, s, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of signed URLs handler", err)
	}

	return nil
}

func (s *SignedUrlHandlerBase) SignUrl(ctx context.Context, originalUrl string, method string) (string, error) {

	u, err := url.Parse(originalUrl)
	if err != nil {
		return "", err
	}

	q := u.Query()
	if s.EXPIRATION != 0 {
		now := time.Now()
		expireAt := now.Add(time.Duration(s.EXPIRATION) * time.Second)
		q.Set(s.EXPIRY_PARAM, strconv.FormatInt(expireAt.Unix(), 10))
	}

	h := s.calcHmac(u, method)
	q.Set(s.SIGNATURE_PARAM, h)

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *SignedUrlHandlerBase) CheckUrlString(ctx context.Context, signedUrl string, method string) error {

	u, err := url.Parse(signedUrl)
	if err != nil {
		return err
	}

	return s.CheckUrl(ctx, u, method)
}

func (s *SignedUrlHandlerBase) CheckUrl(ctx context.Context, u *url.URL, method string) error {

	q := u.Query()

	// check expiration
	expiryStr := q.Get(s.EXPIRY_PARAM)
	if expiryStr != "" {
		expiry, _ := strconv.ParseInt(expiryStr, 10, 64)
		if time.Now().Unix() > expiry {
			return errors.New("expired")
		}
	}

	signature := q.Get(s.SIGNATURE_PARAM)
	return s.checkHmac(u, method, expiryStr, signature)
}

func (s *SignedUrlHandlerBase) calcHmac(u *url.URL, method string) string {

	query := u.Query()
	expiryStr := query.Get("expiry")

	hmac := s.hmacBuilder(s.SECRET)
	h := hmac.CalcStringsStr(method, "\n", expiryStr, "\n", u.Path)
	return h
}

func (s *SignedUrlHandlerBase) checkHmac(u *url.URL, method string, expiry string, signature string) error {
	hmac := s.hmacBuilder(s.SECRET)
	hmac.CalcStrings(method, "\n", expiry, "\n", u.Path)
	return hmac.CheckStr(signature)
}
