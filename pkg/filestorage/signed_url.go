package filestorage

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
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

// Expiration implements SignedUrlHandler - see that interface's doc comment. Mirrors the
// `if s.EXPIRATION != 0` guard in SignUrl(): 0 means no expiry parameter is written into the
// signed URL at all, so there is no absolute expiry to publish either.
func (s *SignedUrlHandlerBase) Expiration() uint32 {
	return s.EXPIRATION
}

func (s *SignedUrlHandlerBase) Init(app app_context.Context, parentConfigPath string, configPath ...string) error {

	path := utils.OptionalString(object_config.Key(parentConfigPath, "signed_url"), configPath...)
	err := object_config.LoadLogValidateApp(app, s, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of signed URLs handler", err)
	}

	return nil
}

func (s *SignedUrlHandlerBase) SignUrl(ctx context.Context, originalUrl string, parameters SignUrlParameters) (string, error) {

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
	// Write the expiry into u.RawQuery *before* computing the HMAC: calcHmac reads the
	// expiry back out of u.Query(), so the signature must be computed over the same
	// RawQuery that CheckUrl will later parse - otherwise the expiry is always signed as
	// empty while CheckUrl verifies against the real value, and every signed URL fails.
	u.RawQuery = q.Encode()

	h := s.calcHmac(u, parameters)
	q.Set(s.SIGNATURE_PARAM, h)

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *SignedUrlHandlerBase) CheckUrlString(ctx context.Context, signedUrl string, parameters SignUrlParameters) error {

	u, err := url.Parse(signedUrl)
	if err != nil {
		return err
	}

	return s.CheckUrl(ctx, u, parameters)
}

func (s *SignedUrlHandlerBase) CheckUrl(ctx context.Context, u *url.URL, parameters SignUrlParameters) error {

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
	return s.checkHmac(u, parameters, expiryStr, signature)
}

func (s *SignedUrlHandlerBase) calcHmac(u *url.URL, parameters SignUrlParameters) string {

	query := u.Query()
	expiryStr := query.Get(s.EXPIRY_PARAM)

	hmac := s.hmacBuilder(s.SECRET)
	h := hmac.CalcStringsStr(strings.Join(parameters.Values(), "\n"), "\n", expiryStr, "\n", u.Path)
	return h
}

func (s *SignedUrlHandlerBase) checkHmac(u *url.URL, parameters SignUrlParameters, expiry string, signature string) error {
	hmac := s.hmacBuilder(s.SECRET)
	hmac.CalcStrings(strings.Join(parameters.Values(), "\n"), "\n", expiry, "\n", u.Path)
	return hmac.CheckStr(signature)
}
