package filestorage

import (
	"context"
	"net/url"
	"strconv"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/crypt_utils"
	"github.com/evgeniums/evgo/pkg/multitenancy"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type UrlManagerConfig struct {
	BASE_UPLOAD_URL   string `validate:"required,url"`
	BASE_DOWNLOAD_URL string `validate:"required,url"`

	UPLOAD_PATH_PREFIX   string `validate:"required" default:"/filedata/upload"`
	DOWNLOAD_PATH_PREFIX string `validate:"required" default:"/filedata/fetch"`

	UPLOAD_METHOD   string `validate:"required,oneof=POST PUT" default:"POST"`
	DOWNLOAD_METHOD string `validate:"required,oneof=GET" default:"GET"`

	ID_PARAMETER      string `validate:"required,alphanum" default:"id"`
	PART_PARAMETER    string `validate:"required,alphanum" default:"part"`
	TENANCY_PARAMETER string `validate:"omitempty,alphanum" default:"tenancy"`
	TOPIC_PARAMETER   string `validate:"omitempty,alphanum" default:"topic"`

	SHADOW_TENANCY_PATH bool `default:"true"`
	ENABLE_TOPIC        bool `default:"false"`
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
	path := utils.OptionalString(object_config.Key(parentConfigPath, "url_manager"), configPath...)
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
			return err
		}
	}

	// done
	return nil
}

func (u *UrlManagerBase) TenancyPath(ctx context.Context) string {
	tenancyCtx := op_context.OpContext[multitenancy.TenancyContext](ctx)
	if tenancyCtx != nil {
		if u.SHADOW_TENANCY_PATH {
			return tenancyCtx.GetTenancy().ShadowPath()
		}
		return tenancyCtx.GetTenancy().Path()
	}
	return ""
}

func (u *UrlManagerBase) GetUploadUrls(ctx context.Context, info FileInfo, fromPartIndex ...int64) (*UploadUrlInfo, error) {

	// prepare result
	result := &UploadUrlInfo{
		TotalUrlCount: u.helper.PartCount(info),
		FromPartIndex: utils.OptionalArg(int64(0), fromPartIndex...),
		Method:        u.UPLOAD_METHOD,
	}
	result.Urls = make([]string, 0, result.TotalUrlCount)
	result.MaxPartLength = u.helper.UploadPartLength(info, 0)

	tenancyPath := u.TenancyPath(ctx)

	// generate urls
	for i := range result.TotalUrlCount {
		var originalUrl string
		var err error
		if tenancyPath == "" {
			if info.GetTopic() == "" {
				originalUrl, err = url.JoinPath(u.BASE_UPLOAD_URL, u.UPLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID(), u.PART_PARAMETER, strconv.FormatInt(i, 10))
			} else {
				originalUrl, err = url.JoinPath(u.BASE_UPLOAD_URL, u.UPLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID(), u.PART_PARAMETER, strconv.FormatInt(i, 10))
			}
		} else {
			if info.GetTopic() == "" {
				originalUrl, err = url.JoinPath(u.BASE_UPLOAD_URL, u.TENANCY_PARAMETER, tenancyPath, u.UPLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID(), u.PART_PARAMETER, strconv.FormatInt(i, 10))
			} else {
				originalUrl, err = url.JoinPath(u.BASE_UPLOAD_URL, u.TENANCY_PARAMETER, tenancyPath, u.UPLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID(), u.PART_PARAMETER, strconv.FormatInt(i, 10))
			}
		}

		if err != nil {
			return nil, err
		}

		v := SignUrlValues{Method: result.Method, ContentLength: strconv.FormatInt(u.helper.UploadPartLength(info, i), 10)}
		signedUrl, err := u.signedUrlHandler.SignUrl(ctx, originalUrl, &v)
		if err != nil {
			return nil, err
		}

		result.Urls = append(result.Urls, signedUrl)
	}

	// done
	return result, nil
}

func (u *UrlManagerBase) Helper() UploadPartHelper {
	return u.helper
}

func (u *UrlManagerBase) SignedUrlHandler() SignedUrlHandler {
	return u.signedUrlHandler
}

func (u *UrlManagerBase) GetDownloadUrl(ctx context.Context, info FileInfo) (*DownloadUrlInfo, error) {

	tenancyPath := u.TenancyPath(ctx)
	var originalUrl string
	var err error

	if tenancyPath == "" {
		if info.GetTopic() == "" {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.DOWNLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID())
		} else {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.DOWNLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID())
		}
	} else {
		if info.GetTopic() == "" {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.TENANCY_PARAMETER, tenancyPath, u.DOWNLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID())
		} else {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.TENANCY_PARAMETER, tenancyPath, u.DOWNLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID())
		}
	}
	if err != nil {
		return nil, err
	}

	resp := &DownloadUrlInfo{}
	v := SignUrlValues{Method: u.DOWNLOAD_METHOD, ContentLength: strconv.FormatInt(info.GetSize(), 10)}
	resp.Url, err = u.signedUrlHandler.SignUrl(ctx, originalUrl, &v)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (u *UrlManagerBase) IdUrlPathParameter() string {
	return u.ID_PARAMETER
}

func (u *UrlManagerBase) PartUrlPathParameter() string {
	return u.PART_PARAMETER
}

func (u *UrlManagerBase) TopicUrlParameter() string {
	return u.TOPIC_PARAMETER
}

func (u *UrlManagerBase) IsTopicEnabled() bool {
	return u.ENABLE_TOPIC
}
