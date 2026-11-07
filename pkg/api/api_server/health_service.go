package api_server

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/evgeniums/evgo/pkg/access_control"
	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/auth"
	"github.com/evgeniums/evgo/pkg/db"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/op_context"
)

const (
	HealthStatusRunning = "running"
	HealthStatusReady   = "ready"

	defaultHealthPingTimeout = 2 * time.Second
	defaultHealthCacheTtl    = time.Second
)

// HealthServiceConfig tunes the readiness check. Zero values select defaults.
type HealthServiceConfig struct {
	// Maximum time to wait for the database ping. Default: 2s.
	PingTimeout time.Duration
	// How long a readiness result is reused. The endpoints are public, so this bounds the
	// database load an anonymous caller can generate. Default: 1s.
	CacheTtl time.Duration
}

// readinessChecker pings the database at most once per CacheTtl. The mutex is held across the
// ping on purpose: concurrent probes coalesce into a single database round trip.
type readinessChecker struct {
	db      db.DB
	timeout time.Duration
	ttl     time.Duration

	mu        sync.Mutex
	checkedAt time.Time
	lastErr   error
}

func (r *readinessChecker) check() error {
	if r.db == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.checkedAt.IsZero() && time.Since(r.checkedAt) < r.ttl {
		return r.lastErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	r.lastErr = r.db.Ping(ctx)
	r.checkedAt = time.Now()
	return r.lastErr
}

// HealthLiveEndpoint reports that the process is up and serving requests. It performs no I/O.
type HealthLiveEndpoint struct {
	ResourceEndpoint
}

func NewHealthLiveEndpoint() *HealthLiveEndpoint {
	ep := &HealthLiveEndpoint{}
	InitResourceEndpoint(ep, "live", "Live", access_control.Get)
	return ep
}

func (e *HealthLiveEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {
	request := op_context.OpContext[Request](sctx)
	request.Response().SetMessage(&StatusResponse{Status: HealthStatusRunning})
	return sctx, nil
}

// HealthReadyEndpoint reports whether the service can do useful work, i.e. its database responds.
type HealthReadyEndpoint struct {
	ResourceEndpoint
	checker *readinessChecker
}

func NewHealthReadyEndpoint(checker *readinessChecker) *HealthReadyEndpoint {
	ep := &HealthReadyEndpoint{checker: checker}
	InitResourceEndpoint(ep, "ready", "Ready", access_control.Get)
	return ep
}

func (e *HealthReadyEndpoint) HandleRequest(sctx context.Context) (context.Context, error) {
	request := op_context.OpContext[Request](sctx)

	err := e.checker.check()
	if err != nil {
		// The cause is returned to the framework (logged) but never put into the response:
		// the endpoint is public.
		request.SetGenericErrorCode(generic_error.ErrorCodeUnavailable)
		return sctx, err
	}

	request.Response().SetMessage(&StatusResponse{Status: HealthStatusReady})
	return sctx, nil
}

// HealthService is a minimal, public service for external monitoring and load balancers:
//
//	GET <prefix>/<version>/health/live   - process is up
//	GET <prefix>/<version>/health/ready  - process is up and the database responds (503 otherwise)
type HealthService struct {
	ServiceBase
}

func NewHealthService(database db.DB, cfg ...HealthServiceConfig) *HealthService {

	c := HealthServiceConfig{}
	if len(cfg) != 0 {
		c = cfg[0]
	}
	if c.PingTimeout <= 0 {
		c.PingTimeout = defaultHealthPingTimeout
	}
	if c.CacheTtl <= 0 {
		c.CacheTtl = defaultHealthCacheTtl
	}

	s := &HealthService{}
	s.Init("health", api.PackageName)
	s.AddChildren(
		NewHealthLiveEndpoint(),
		NewHealthReadyEndpoint(&readinessChecker{db: database, timeout: c.PingTimeout, ttl: c.CacheTtl}),
	)
	return s
}

// AddHealthService registers HealthService on a REST (Gin) server and makes its endpoints public
// without any configuration: the "noauth" method is registered if the auth configuration lacks it,
// and only the two health paths are mapped to it, so the default auth schema is unchanged.
//
// REST servers only: the responses are plain structs that cannot be marshalled as protobuf, so
// do not call this with a gRPC server.
func AddHealthService(server Server, database db.DB, cfg ...HealthServiceConfig) error {

	endpointsAuth, ok := server.Auth().(auth.EndpointsAuth)
	if !ok {
		return errors.New("health service requires auth with per-endpoint schema configuration")
	}

	// make sure that "noauth" method is available
	manager := endpointsAuth.Manager()
	if _, err := manager.Handlers().Handler(auth.NoAuthProtocol); err != nil {
		noAuth := &auth.NoAuthMethod{}
		if err := noAuth.Init(nil, nil, nil); err != nil {
			return err
		}
		manager.Handlers().AddHandler(noAuth)
		noAuth.SetAuthManager(manager)
	}

	service := NewHealthService(database, cfg...)

	// map the health paths to "noauth"; the key is the same one the REST server passes to Auth
	err := service.EachOperation(func(op api.Operation) error {
		ep, ok := op.(Endpoint)
		if !ok {
			return errors.New("invalid health operation type, must be endpoint: " + op.Name())
		}
		endpointsAuth.EndpointsConfig().AddSchema(ep.Resource().ServicePathPrototype(), access_control.Get, auth.NoAuthProtocol)
		return nil
	})
	if err != nil {
		return err
	}

	// AddServiceToServer attaches the service, which registers its endpoints on the server
	return AddServiceToServer(server, service)
}
