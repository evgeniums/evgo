package test_utils

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/api/api_server/rest_api_gin_server"
	"github.com/evgeniums/evgo/pkg/api/bare_bones_server"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func BBRestApiServer(t *testing.T, server bare_bones_server.Server) *rest_api_gin_server.Server {
	apiServer := server.ApiServer()
	restApiServer, ok := apiServer.(*rest_api_gin_server.Server)
	require.True(t, ok, "API server is not a REST API server")
	require.NotNil(t, restApiServer)
	return restApiServer
}

func BBGinEngine(t *testing.T, server bare_bones_server.Server) *gin.Engine {
	restApiServer := BBRestApiServer(t, server)
	return restApiServer.GinEngine()
}
