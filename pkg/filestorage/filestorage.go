package filestorage

import (
	"context"
	"io"
	"net/url"

	"github.com/evgeniums/evgo/pkg/common"
	"github.com/evgeniums/evgo/pkg/utils"
)

type FileInfo interface {
	common.Object

	GetContentType() string
	GetSize() int64
	GetFileName() string

	GetTopic() string

	GetNativeId() string
	GetUploadPartSize() int64
}

type SignUrlParameters interface {
	Values() []string
}

type SignUrlValues struct {
	Method        string
	ContentLength string
}

func (s *SignUrlValues) Values() []string {
	if s.ContentLength == "" {
		return []string{s.Method}
	}
	return []string{s.Method, s.ContentLength}
}

type SignedUrlHandler interface {
	SignUrl(ctx context.Context, originalUrl string, parameters SignUrlParameters) (string, error)
	CheckUrlString(ctx context.Context, signedUrl string, parameters SignUrlParameters) error

	CheckUrl(ctx context.Context, signedUrl *url.URL, parameters SignUrlParameters) error

	// Expiration is the lifetime in seconds a URL signed by this handler is granted, or 0 if
	// signed URLs never expire. Exposed so UrlManager can publish an absolute expiry to the
	// client (UploadUrlInfo.UrlExpiresAt) instead of leaving the client to parse it back out
	// of the signed URL - see that field's doc comment for why the client must not infer it.
	Expiration() uint32
}

type UploadUrlInfo struct {
	Urls             []string `json:"urls"`
	TotalUrlCount    int64    `json:"total_url_count"`
	Method           string   `json:"method"`
	FromPartIndex    int64    `json:"from_part_index"`
	CertificateChain string   `json:"certificate_chain"`
	ProxyHeader      string   `json:"proxy_header"`
	ProxyHeaderName  string   `json:"proxy_header_name"`
	MaxPartLength    int64    `json:"max_part_length"`

	// UseSystemCa and SkipHostNameVerification are server-declared TLS policy for the file
	// transfer endpoint - see UrlManagerConfig.USE_SYSTEM_CA/SKIP_HOST_NAME_VERIFICATION.
	// Both default false: a client trusts only CertificateChain and always verifies the peer
	// name against the URL host unless the server explicitly relaxes either check.
	UseSystemCa              bool `json:"use_system_ca"`
	SkipHostNameVerification bool `json:"skip_host_name_verification"`

	// UrlExpiresAt is the absolute unix time (seconds) at which Urls stop being accepted, or 0
	// if they never expire. Stated by the server so the client never has to parse expiry out of
	// the URL - see the proto field's doc comment for the full rationale.
	UrlExpiresAt int64 `json:"url_expires_at"`
}

type DownloadUrlInfo struct {
	Url              string `json:"url"`
	CertificateChain string `json:"certificate_chain"`
	ProxyHeader      string `json:"proxy_header"`
	ProxyHeaderName  string `json:"proxy_header_name"`

	// See UploadUrlInfo.UseSystemCa/SkipHostNameVerification - same meaning and defaults.
	UseSystemCa              bool `json:"use_system_ca"`
	SkipHostNameVerification bool `json:"skip_host_name_verification"`

	// See UploadUrlInfo.UrlExpiresAt - same meaning, for the single Url above.
	UrlExpiresAt int64 `json:"url_expires_at"`
}

type UploadPartHelper interface {
	UploadPartLength(info FileInfo, partIndex ...int64) int64
	PartCount(info FileInfo) int64
}

type GetUploadUrlsOptions struct {
	FromPart int64
	MaxCount int64
}

type UrlManager interface {
	GetUploadUrls(ctx context.Context, info FileInfo, opt ...GetUploadUrlsOptions) (*UploadUrlInfo, error)
	GetDownloadUrl(ctx context.Context, info FileInfo) (*DownloadUrlInfo, error)

	IdUrlPathParameter() string
	PartUrlPathParameter() string
	TopicUrlParameter() string
	IsTopicEnabled() bool
}

type StorageManager interface {
	StartUpload(ctx context.Context, info FileInfo) error
	UploadPart(ctx context.Context, info FileInfo, source io.Reader, partIndex ...int64) error
	FinalizeUpload(ctx context.Context, info FileInfo, partsCount ...int64) error

	Fetch(ctx context.Context, info FileInfo, offset ...int64) (io.ReadCloser, error)
	FetchRange(ctx context.Context, info FileInfo, offset int64, length int64) (io.ReadCloser, error)

	// Exists reports whether the FINALIZED content of info is present, addressed exactly the
	// way Path()/Fetch()/DeleteFile() resolve it (tenancy/topic aware). false with a nil error
	// means "definitely not there"; a non-nil error means the backend could not answer and the
	// caller must not treat that as absence.
	Exists(ctx context.Context, info FileInfo) (bool, error)

	Delete(ctx context.Context, pathPrefix string) error

	// DeleteFile removes the finalized content of info, addressed the same way Path()/Fetch()
	// resolve it (tenancy/topic aware). A no-op (not an error) if the content was never
	// finalized or was already removed, so callers can use it idempotently.
	DeleteFile(ctx context.Context, info FileInfo) error

	DeleteTemp(ctx context.Context, toDate utils.Date) error
}

type FileInfoRegistry interface {
	FindForUpload(ctx context.Context, id string, part int64, topic ...string) (FileInfo, error)
	FindForDownload(ctx context.Context, id string, topic ...string) (FileInfo, error)
}
