package rest_api_client

import (
	"net/http"

	"github.com/evgeniums/go-utils/pkg/access_control"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/auth"
	"github.com/evgeniums/go-utils/pkg/auth/auth_methods/auth_login_phash"
	"github.com/evgeniums/go-utils/pkg/generic_error"
	"github.com/evgeniums/go-utils/pkg/multitenancy"
	"github.com/evgeniums/go-utils/pkg/op_context"
	"github.com/evgeniums/go-utils/pkg/utils"
)

type Auth interface {
	MakeHeaders(ctx op_context.Context, operation api.Operation, cmd interface{}) (map[string]string, error)
	HandleResponse(resp Response)
}

type RestApiMethod func(ctx op_context.Context, path string, cmd interface{}, response interface{}, headers ...map[string]string) (Response, error)

type Client struct {
	RestApiClient      RestApiClient
	methods            map[access_control.AccessType]RestApiMethod
	auth               Auth
	propagateAuthUser  bool
	propagateContextId bool
}

func New(restApiClient RestApiClient, auth ...Auth) *Client {
	c := &Client{RestApiClient: restApiClient}
	c.methods = make(map[access_control.AccessType]RestApiMethod, 0)

	c.methods[access_control.HttpMethod2Access(http.MethodPost)] = c.RestApiClient.Post
	c.methods[access_control.HttpMethod2Access(http.MethodPut)] = c.RestApiClient.Put
	c.methods[access_control.HttpMethod2Access(http.MethodPatch)] = c.RestApiClient.Patch
	c.methods[access_control.HttpMethod2Access(http.MethodGet)] = c.RestApiClient.Get
	c.methods[access_control.HttpMethod2Access(http.MethodDelete)] = c.RestApiClient.Delete

	if len(auth) != 0 {
		c.auth = auth[0]
	}

	return c
}

func (cl *Client) Transport() interface{} {
	return cl.RestApiClient
}

func (cl *Client) SetPropagateAuthUser(val bool) {
	cl.propagateAuthUser = true
}

func (cl *Client) SetPropagateContextId(val bool) {
	cl.propagateContextId = true
}

func (cl *Client) Exec(ctx op_context.Context, operation api.Operation, cmd interface{}, response interface{}, tenancyArg ...multitenancy.TenancyPath) error {

	// TODO support hateoas links of resource

	// setup
	c := ctx.TraceInMethod("Client.Exec")
	defer ctx.TraceOutMethod()

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

	// find method for operation
	method, ok := cl.methods[operation.AccessType()]
	if !ok {
		c.SetLoggerField("access_type", operation.AccessType())
		genericError := generic_error.NewFromMessage("access type not supported")
		c.SetError(genericError)
		return genericError
	}

	// evaluate path
	var path string
	tenancy := utils.OptionalArg(nil, tenancyArg...)
	if tenancy == nil {
		path = operation.Resource().FullActualPath()
	} else {
		path = operation.Resource().FullActualTenancyPath(tenancy.Path())
	}

	var resp Response
	var err error
	var errr error
	if cl.auth != nil {

		// c.Logger().Debug("invoke with auth")

		exec := func() {
			// make auth headers
			headers, err1 := cl.auth.MakeHeaders(ctx, operation, cmd)
			if err1 != nil {
				c.SetMessage("failed to make auth headers")
				errr = err1
			}
			if forwardHeaders != nil {
				utils.AppendMap(headers, forwardHeaders)
			}
			// invoke method with auth headers
			resp, err = method(ctx, path, cmd, response, headers)
			cl.auth.HandleResponse(resp)
		}
		exec()
		if errr != nil {
			// c.Logger().Debug("auth headers failed")
			return c.SetError(errr)
		}
		if resp != nil && resp.Code() == http.StatusUnauthorized && !auth_login_phash.IsLoginError(resp.Error()) {
			exec()
			if errr != nil {
				// c.Logger().Debug("second auth headers failed")
				return c.SetError(errr)
			}
		}

	} else if forwardHeaders != nil {
		// invoke method with context auth user
		resp, err = method(ctx, path, cmd, response, forwardHeaders)
	} else {
		// invoke method without auth headers
		resp, err = method(ctx, path, cmd, response)
	}

	// process generic error
	if resp != nil {
		// c.Logger().Debug("resp not nil")
		genericError := resp.Error()
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
		c.SetMessage("failed to invoke HTTP method")
		return c.SetError(err)
	}

	// done
	// c.Logger().Debug("exex ok")
	return nil
}
