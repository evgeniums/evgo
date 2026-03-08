package api_client

import "context"

type AutoReconnectHandlers interface {
	GetRefreshToken() string
	SaveRefreshToken(sctx context.Context, token string)
	GetCredentials(sctx context.Context) (login string, password string, err error)
}
