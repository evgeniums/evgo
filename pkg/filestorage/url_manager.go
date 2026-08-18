package filestorage

import (
	"context"
	"net/url"
	"os"
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

	// CERTIFICATE_CHAIN is an optional PEM file for the storage endpoint's own CA - needed only
	// when BASE_UPLOAD_URL/BASE_DOWNLOAD_URL point at a host (CDN, separate storage node) using a
	// different certificate than the gRPC route host. When unset, callers fall back to the route
	// host certificate (see file_controller.ControllerBase.applyTlsPolicy).
	CERTIFICATE_CHAIN string `validate:"omitempty,file" vmessage:"Invalid file path"`
	// USE_SYSTEM_CA and SKIP_HOST_NAME_VERIFICATION are published to clients verbatim as
	// UploadUrlInfo/DownloadUrlInfo TLS policy - see those types' doc comments for the
	// false-by-default rationale.
	USE_SYSTEM_CA               bool
	SKIP_HOST_NAME_VERIFICATION bool
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
	certificateChain []byte
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

	if u.CERTIFICATE_CHAIN != "" {
		u.certificateChain, err = os.ReadFile(u.CERTIFICATE_CHAIN)
		if err != nil {
			return app.Logger().PushFatalStack("failed to read url manager certificate chain file", err)
		}
	}

	// init helper
	u.helper = opt.Helper
	if u.helper == nil {
		helper := NewUploadPartHelper()
		err = helper.Init(app, path, path)
		if err != nil {
			return err
		}
		u.helper = helper
	}

	// init URL handler
	u.signedUrlHandler = opt.SignedUrlHandler
	if u.signedUrlHandler == nil {
		signedUrlHandler := NewSignedUrl(opt.HmacBuilder)
		err = signedUrlHandler.Init(app, path)
		if err != nil {
			return err
		}
		u.signedUrlHandler = signedUrlHandler
	}

	// done
	return nil
}

func (u *UrlManagerBase) TenancyPath(ctx context.Context) string {
	tenancyCtx := op_context.OpContext[multitenancy.TenancyContext](ctx)
	if tenancyCtx != nil && tenancyCtx.GetTenancy() != nil {
		if u.SHADOW_TENANCY_PATH {
			return tenancyCtx.GetTenancy().ShadowPath()
		}
		return tenancyCtx.GetTenancy().Path()
	}
	return ""
}

func (u *UrlManagerBase) GetUploadUrls(ctx context.Context, info FileInfo, opt ...GetUploadUrlsOptions) (*UploadUrlInfo, error) {

	fromPartIndex := int64(0)
	maxCount := int64(0)
	if len(opt) > 0 {
		fromPartIndex = opt[0].FromPart
		maxCount = opt[0].MaxCount
	}

	// prepare result
	result := &UploadUrlInfo{
		TotalUrlCount:            u.helper.PartCount(info),
		FromPartIndex:            fromPartIndex,
		Method:                   u.UPLOAD_METHOD,
		CertificateChain:         string(u.certificateChain),
		UseSystemCa:              u.USE_SYSTEM_CA,
		SkipHostNameVerification: u.SKIP_HOST_NAME_VERIFICATION,
	}
	result.Urls = make([]string, 0, result.TotalUrlCount)
	result.MaxPartLength = u.helper.UploadPartLength(info, 0)

	tenancyPath := u.TenancyPath(ctx)

	// generate urls
	toPartIndex := result.TotalUrlCount
	if maxCount != 0 {
		toPartIndex = fromPartIndex + maxCount
		if toPartIndex > result.TotalUrlCount {
			toPartIndex = result.TotalUrlCount
		}
	}
	// Task debug-sending-files-to-optimized-music (server stage 2, C): the
	// topic segment must be gated on u.ENABLE_TOPIC, the SAME switch
	// filedata_service.go uses to decide whether the route itself has a
	// /topic/:topic segment - gating on info.GetTopic()!="" alone (the
	// previous condition) generated a URL with a topic segment regardless of
	// ENABLE_TOPIC, 404ing against a route that was never registered to
	// expect one.
	includeTopic := u.ENABLE_TOPIC && info.GetTopic() != ""

	for i := fromPartIndex; i < toPartIndex; i++ {
		var originalUrl string
		var err error
		if tenancyPath == "" {
			if !includeTopic {
				originalUrl, err = url.JoinPath(u.BASE_UPLOAD_URL, u.UPLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID(), u.PART_PARAMETER, strconv.FormatInt(i, 10))
			} else {
				originalUrl, err = url.JoinPath(u.BASE_UPLOAD_URL, u.UPLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID(), u.PART_PARAMETER, strconv.FormatInt(i, 10))
			}
		} else {
			if !includeTopic {
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

	// See GetUploadUrls()'s identical comment on why ENABLE_TOPIC, not just
	// info.GetTopic()!="", gates the topic segment.
	includeTopic := u.ENABLE_TOPIC && info.GetTopic() != ""

	if tenancyPath == "" {
		if !includeTopic {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.DOWNLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID())
		} else {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.DOWNLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID())
		}
	} else {
		if !includeTopic {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.TENANCY_PARAMETER, tenancyPath, u.DOWNLOAD_PATH_PREFIX, u.ID_PARAMETER, info.GetID())
		} else {
			originalUrl, err = url.JoinPath(u.BASE_DOWNLOAD_URL, u.TENANCY_PARAMETER, tenancyPath, u.DOWNLOAD_PATH_PREFIX, u.TOPIC_PARAMETER, info.GetTopic(), u.ID_PARAMETER, info.GetID())
		}
	}
	if err != nil {
		return nil, err
	}

	resp := &DownloadUrlInfo{
		CertificateChain:         string(u.certificateChain),
		UseSystemCa:              u.USE_SYSTEM_CA,
		SkipHostNameVerification: u.SKIP_HOST_NAME_VERIFICATION,
	}
	// ContentLength is deliberately omitted here: FileDataControllerBase.checkUrl()
	// (filedata_service/filedata_controller.go) only sets it in the verification
	// SignUrlValues for uploads (`if !download {...}`) - a download's byte count isn't
	// chosen by the client the way an upload part's is, so there is nothing to commit
	// to. Signing with it anyway meant SignUrl and CheckUrl always disagreed on the
	// value set whenever EXPIRATION!=0 (the shipped default), so every signed download
	// URL failed verification unconditionally - found while implementing the files2
	// client download queue (task 5).
	v := SignUrlValues{Method: u.DOWNLOAD_METHOD}
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
