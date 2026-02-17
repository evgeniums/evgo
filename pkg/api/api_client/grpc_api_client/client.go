package grpc_api_client

import (
	"net/http"

	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/logger"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/utils"
	"google.golang.org/grpc"
)

type OperationResponse interface {
	api_client.OperationResponse
}

type GrpcResponse struct {
	code        int
	serverError generic_error.Error
}

func (r *GrpcResponse) Code() int {
	return r.code
}

func (r *GrpcResponse) Error() generic_error.Error {
	return r.serverError
}

func (r *GrpcResponse) SetError(err generic_error.Error) {
	r.serverError = err
}

func IsResponseOK(resp OperationResponse, err error) bool {
	if err != nil || resp == nil {
		return false
	}
	return resp.Code() < http.StatusBadRequest
}

type Auth interface {
	MakeHeaders(ctx op_context.Context, operation api.Operation, cmd interface{}) (map[string]string, error)
	HandleResponse(resp OperationResponse)
}

type GrpcClient interface {
	GrpcConnection() *grpc.ClientConn
}

type ClientServices interface {
	Init(client GrpcClient) error
}

type Client[T ClientServices] struct {
	services           T
	auth               Auth
	propagateAuthUser  bool
	propagateContextId bool

	grpcConn *grpc.ClientConn
}

func New[T ClientServices](pbServices T) *Client[T] {
	return &Client[T]{services: pbServices}
}

func (cl *Client[T]) Init(log logger.Logger, target string, opts []grpc.DialOption, auth ...Auth) error {

	// setup auth
	if len(auth) != 0 {
		cl.auth = auth[0]
	}

	// init connection
	var err error
	cl.grpcConn, err = grpc.NewClient(target, opts...)
	if err != nil {
		return log.PushFatalStack("failed to create grpc client", err)
	}

	// init services
	err = cl.services.Init(cl)
	if err != nil {
		return log.PushFatalStack("failed to init grpc services", err)
	}

	// done
	return nil

}

func (cl *Client[T]) Services() T {
	return cl.services
}

func (cl *Client[T]) GrpcConnection() *grpc.ClientConn {
	return cl.grpcConn
}

func (cl *Client[T]) Transport() interface{} {
	return cl.grpcConn
}

func (cl *Client[T]) SetPropagateAuthUser(val bool) {
	cl.propagateAuthUser = true
}

func (cl *Client[T]) SetPropagateContextId(val bool) {
	cl.propagateContextId = true
}

func (cl *Client[T]) Exec(ctx op_context.Context, operation api.Operation, requestMessage interface{}, resultMessage interface{}, tenancyArg ...multitenancy.TenancyPath) error {

	c := ctx.TraceInMethod("Client.Exec")
	defer ctx.TraceOutMethod()

	// fill forward headers propagated in context from other microservice
	var forwardHeaders map[string]string
	if cl.propagateContextId {
		forwardHeaders = make(map[string]string)
		forwardHeaders[api.ForwardContext] = ctx.ID()
		if ctx.Origin() != nil {
			forwardHeaders[api.ForwardOpSource] = ctx.Origin().Source()
			forwardHeaders[api.ForwardSessionClient] = ctx.Origin().SessionClient()
		}
	}
	if cl.propagateAuthUser {
		authUserCtx, ok := ctx.(auth.ContextWithAuthUser)
		if ok {
			authUser := authUserCtx.AuthUser()
			if authUser != nil {
				if forwardHeaders == nil {
					forwardHeaders = make(map[string]string)
				}
				forwardHeaders[api.ForwardUserLogin] = authUser.Login()
				forwardHeaders[api.ForwardUserDisplay] = authUser.Display()
				forwardHeaders[api.ForwardUserId] = authUser.GetID()
			}
		}
	}

	var opResp OperationResponse
	var err error
	var errr error

	opExec := func(headers ...api.OperationHeaders) {
		var h api.OperationHeaders
		if len(headers) > 0 {
			h = headers[0]
		}
		var tmpResp interface{}
		tmpResp, err = operation.Exec(ctx, cl, requestMessage, resultMessage, h, tenancyArg...)
		opResp = tmpResp.(OperationResponse)
	}

	if cl.auth != nil {

		// invoke operation with auth

		// c.Logger().Debug("invoke with auth")

		exec := func() {
			// make auth headers
			headers, err1 := cl.auth.MakeHeaders(ctx, operation, requestMessage)
			if err1 != nil {
				c.SetMessage("failed to make auth headers")
				errr = err1
			}
			if forwardHeaders != nil {
				utils.AppendMap(headers, forwardHeaders)
			}
			// invoke method with auth headers
			opExec(headers)
			cl.auth.HandleResponse(opResp)
		}
		exec()
		if errr != nil {
			// c.Logger().Debug("auth headers failed")
			return c.SetError(errr)
		}
		if opResp != nil && opResp.Code() == http.StatusUnauthorized && !auth_login_phash.IsLoginError(opResp.Error()) {
			exec()
			if errr != nil {
				// c.Logger().Debug("second auth headers failed")
				return c.SetError(errr)
			}
		}

	} else if forwardHeaders != nil {
		// invoke method with context auth user
		opExec(forwardHeaders)
	} else {
		// invoke method without auth headers
		opExec()
	}

	// process generic error
	if opResp != nil {
		genericError := opResp.Error()
		if genericError != nil {
			// c.Logger().Debug("resp is generic error")
			c.SetLoggerField("response_code", genericError.Code())
			c.SetLoggerField("response_message", genericError.Message())
			c.SetLoggerField("response_details", genericError.Details())
			c.SetMessage("server returned error")
			ctx.SetGenericError(genericError, true)
			return c.SetError(genericError)
		}
	} else {
		// c.Logger().Debug("resp is nil")
	}

	// check error
	if err != nil {
		// c.Logger().Debug("exec failed")
		c.SetMessage("failed to invoke gRPC method")
		return c.SetError(err)
	}

	// done
	// c.Logger().Debug("exec ok")
	return nil
}
