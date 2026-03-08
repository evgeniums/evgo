package http_request

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/evgeniums/evgo/pkg/config"
	"github.com/evgeniums/evgo/pkg/config/object_config"
	"github.com/evgeniums/evgo/pkg/logger"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/evgeniums/evgo/pkg/validator"
)

type HttpClientConfig struct {
	TIMEOUT                  int `default:"15"`
	KEEP_ALIVE               int `default:"30"`
	MAX_IDLE_CONNECTIONS     int `default:"100"`
	IDLE_CONNECTIONS_TIMEOUT int `default:"90"`
	TLS_HANDSHAKE_TIMEOUT    int `default:"10"`
	EXPECT_CONTINUE_TIMEOUT  int `default:"1"`

	USER_AGENT string `default:"sed"`
}

type HttpClient struct {
	HttpClientConfig
	httpClient *http.Client
	context    context.Context
	cancel     context.CancelFunc
	transport  *http.Transport
}

func (c *HttpClient) Config() interface{} {
	return &c.HttpClientConfig
}

func NewHttpClient(transport ...http.RoundTripper) *HttpClient {
	h := &HttpClient{}
	h.httpClient = &http.Client{Transport: utils.OptionalArg(nil, transport...)}
	h.context, h.cancel = context.WithCancel(context.Background())
	return h
}

func DefaultHttpClient() *HttpClient {
	return NewHttpClient(http.DefaultTransport)
}

func transportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}

func (h *HttpClient) Init(cfg config.Config, log logger.Logger, vld validator.Validator, configPath ...string) error {

	var err error

	if h.httpClient.Transport != nil {
		log.Info("Configuration: using transport from constructor")
		return nil
	}

	err = object_config.LoadLogValidate(cfg, log, vld, h, "http_client")
	key := utils.OptionalString("http_client", configPath...)
	if cfg.IsSet(key) {
		err = object_config.LoadLogValidate(cfg, log, vld, h, key)
	}
	if err != nil {
		return log.PushFatalStack("failed to load configuration of http client", err)
	}

	h.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: transportDialContext(&net.Dialer{
			Timeout:   time.Duration(h.TIMEOUT) * time.Second,
			KeepAlive: time.Duration(h.KEEP_ALIVE) * time.Second,
		}),
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          h.MAX_IDLE_CONNECTIONS,
		IdleConnTimeout:       time.Duration(h.IDLE_CONNECTIONS_TIMEOUT) * time.Second,
		TLSHandshakeTimeout:   time.Duration(h.TLS_HANDSHAKE_TIMEOUT) * time.Second,
		ExpectContinueTimeout: time.Duration(h.EXPECT_CONTINUE_TIMEOUT) * time.Second,
	}
	h.httpClient.Transport = h.transport
	h.httpClient.Timeout = time.Duration(h.TIMEOUT) * time.Second
	return nil
}

func (h *HttpClient) SetTlsConfig(cfg *tls.Config) {
	if h.transport != nil {
		h.transport.TLSClientConfig = cfg
	}
}

func (h *HttpClient) NewPost(sctx context.Context, url string, msg interface{}, serializer ...message.Serializer) (*Request, error) {
	req, err := NewPostWithContext(h.context, sctx, url, msg, serializer...)
	if err != nil {
		return nil, err
	}
	req.client = h.httpClient
	req.UserAgent = h.USER_AGENT
	return req, nil
}

func (h *HttpClient) NewGet(sctx context.Context, url string, msg interface{}) (*Request, error) {
	req, err := NewGetWithContext(h.context, sctx, url, msg)
	if err != nil {
		return nil, err
	}
	req.client = h.httpClient
	req.UserAgent = h.USER_AGENT
	return req, nil
}

func (h *HttpClient) NewRequest(method, url string, body io.Reader) (*Request, error) {
	var err error
	r := &Request{}
	r.NativeRequest, err = http.NewRequestWithContext(h.context, method, url, body)
	if err != nil {
		return nil, err
	}
	r.client = h.httpClient
	r.UserAgent = h.USER_AGENT
	return r, nil
}

func (h *HttpClient) Shutdown(ctx context.Context) error {
	if h.cancel != nil {
		h.cancel()
	}
	return nil
}

func (h *HttpClient) Context() context.Context {
	return h.context
}

type WithHttpClient struct {
	httpClient *HttpClient
}

func (w *WithHttpClient) Construct(transport ...http.RoundTripper) {
	w.httpClient = NewHttpClient(transport...)
}

func (w *WithHttpClient) HttpClient() *HttpClient {
	return w.httpClient
}

func (w *WithHttpClient) Init(cfg config.Config, log logger.Logger, vld validator.Validator, parentConfigPath string) error {
	httpClientPath := object_config.Key(parentConfigPath, "http_client")
	w.httpClient = NewHttpClient()
	err := w.httpClient.Init(cfg, log, vld, httpClientPath)
	if err != nil {
		return err
	}
	return nil
}

func (w *WithHttpClient) Shutdown(ctx context.Context) error {
	if w.httpClient != nil {
		return w.httpClient.Shutdown(ctx)
	}
	return nil
}
