package pool_misrocervice_client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/evgeniums/evgo/pkg/api/api_client/rest_api_client"
	"github.com/evgeniums/evgo/pkg/http_request"
	"github.com/evgeniums/evgo/pkg/pool"
	"github.com/evgeniums/evgo/pkg/utils"
)

func BaseUrlForService(service *pool.PoolServiceBinding, public ...bool) (string, error) {

	pub := utils.OptionalArg(false, public...)
	url := ""

	if pub {
		url = service.PublicUrl()
		if url == "" {
			host := service.PublicHost()
			if host == "" {
				return "", errors.New("unknown URL of the service")
			}
			if !strings.HasPrefix(host, "http") {
				host = utils.ConcatStrings("https://", host)
			}
			portStr := ""
			if service.PublicPort() != 443 {
				portStr = fmt.Sprintf(":%d", service.PublicPort())
			}
			apiVersion := ""
			if service.ApiVersion() != "" {
				apiVersion = fmt.Sprintf("/%s", service.ApiVersion())
			}
			url = fmt.Sprintf("%s%s%s%s", host, portStr, service.PathPrefix(), apiVersion)
		}
	} else {
		url = service.PrivateUrl()
		if url == "" {
			host := service.PrivateHost()
			if host == "" {
				return "", errors.New("unknown URL of the service")
			}
			if !strings.HasPrefix(host, "http") {
				host = utils.ConcatStrings("http://", host)
			}
			if service.PrivatePort() == 0 {
				return "", errors.New("unknown port of the service")
			}
			apiVersion := ""
			if service.ApiVersion() != "" {
				apiVersion = fmt.Sprintf("/%s", service.ApiVersion())
			}
			url = fmt.Sprintf("%s:%d%s%s", host, service.PrivatePort(), service.PathPrefix(), apiVersion)
		}
	}

	return url, nil
}

type RestApiPoolServiceClient struct {
	*rest_api_client.Client
}

func (r *RestApiPoolServiceClient) InitForPoolService(httpClient *http_request.HttpClient, service *pool.PoolServiceBinding, clientAgent ...string) error {

	url, err := BaseUrlForService(service)
	if err != nil {
		return err
	}

	r.Client = rest_api_client.New(rest_api_client.DefaultRestApiClient(httpClient, url, clientAgent...))

	return nil
}
