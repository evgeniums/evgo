package filestorage

import (
	"context"
	"net/url"
	"strconv"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/utils"
)

type UrlManagerConfig struct {
	BASE_UPLOAD_URL   string `validate:"required,http_url"`
	BASE_DOWNLOAD_URL string `validate:"required,http_url"`
	UPLOAD_METHOD     string `validate:"required,one_of=POST PUT" default:"POST"`
	DOWNLOAD_METHOD   string `validate:"required,one_of=GET" default:"GET"`
}

type UrlManagerOptions struct {
	Helper           UploadPartHelper
	SignedUrlHandler SignedUrlHandler
	HmacBuilder      crypt_utils.HmacBuilder
}

type UrlManagerBase struct {
	UrlManagerConfig
	helper           UploadPartHelper
	signedUrlHandler SignedUrlHandler
}

func NewUrlManager() *UrlManagerBase {
	u := &UrlManagerBase{}
	return u
}

func (u *UrlManagerBase) Config() any {
	return &u.UrlManagerConfig
}

func (u *UrlManagerBase) Init(app app_context.Context, opt UrlManagerOptions, parentConfigPath string, configPath ...string) error {

	// load config
	path := object_config.Key(parentConfigPath, utils.OptionalString("url_manager", configPath...))
	err := object_config.LoadLogValidateApp(app, u, path)
	if err != nil {
		return app.Logger().PushFatalStack("failed to load configuration of URL manager", err)
	}

	// init helper
	u.helper = opt.Helper
	if u.helper == nil {
		helper := NewUploadPartHelper()
		err = helper.Init(app, path, path)
		if err != nil {
			return err
		}
	}

	// init URL handler
	u.signedUrlHandler = opt.SignedUrlHandler
	if u.signedUrlHandler == nil {
		signedUrlHandler := NewSignedUrl(opt.HmacBuilder)
		err = signedUrlHandler.Init(app, path)
		if err != nil {
			return nil
		}
	}

	// done
	return nil
}

func (u *UrlManagerBase) GetUploadUrls(ctx context.Context, info *FileInfo, fromPartIndex ...int64) (*UploadUrlInfo, error) {

	// prepare result
	result := &UploadUrlInfo{
		TotalUrlCount: u.helper.PartCount(info),
		FromPartIndex: utils.OptionalArg(int64(0), fromPartIndex...),
		Method:        u.UPLOAD_METHOD,
	}
	result.Urls = make([]string, 0, result.TotalUrlCount)

	// generate urls
	for i := range result.TotalUrlCount {
		originalUrl, err := url.JoinPath(u.BASE_UPLOAD_URL, info.GetID(), strconv.FormatInt(i, 10))
		if err != nil {
			return nil, err
		}

		signedUrl, err := u.signedUrlHandler.SignUrl(ctx, originalUrl, result.Method)
		if err != nil {
			return nil, err
		}

		result.Urls = append(result.Urls, signedUrl)
	}

	// done
	return result, nil
}

func (u *UrlManagerBase) GetDownloadUrl(ctx context.Context, info *FileInfo) (string, error) {

	originalUrl, err := url.JoinPath(u.BASE_DOWNLOAD_URL, info.GetID())
	if err != nil {
		return "", err
	}

	return u.signedUrlHandler.SignUrl(ctx, originalUrl, u.DOWNLOAD_METHOD)
}
