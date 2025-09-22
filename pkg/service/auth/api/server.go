package api

import (
	"context"

	// gin
	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/tka/internal/utils"
	client "github.com/spechtlabs/tka/pkg/client/k8s"
	mw "github.com/spechtlabs/tka/pkg/middleware"
	authMw "github.com/spechtlabs/tka/pkg/middleware/auth"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"tailscale.com/tailcfg"

	// Misc
	"github.com/sierrasoftworks/humane-errors-go"

	// Logging

	// o11y
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	// tka
	ts "github.com/spechtlabs/tka/pkg/tailscale"
)

const (
	ApiRouteV1Alpha1   = "/api/v1alpha1"
	LoginApiRoute      = "/login"
	KubeconfigApiRoute = "/kubeconfig"
	LogoutApiRoute     = "/logout"
)

// TKAServer represents the main HTTP server for Tailscale Kubernetes Auth.
// It provides a complete HTTP API for user authentication, credential management,
// and cluster operations over a Tailscale network.
//
// The server consists of the following components:
// - Gin HTTP router with OpenTelemetry observability middlewares (ginzap, otelgin, prometheus)
//   - Otel Tracing
//   - Prometheus metrics
//   - Ginzap logging
//
// - Gin authentication middleware using the tailscale.WhoIsResolver
// - Service layer for business logic
// - Swagger documentation endpoint
//
// 1. Create server with NewTKAServer() constructor
// 2. Load routes with LoadApiRoutes() and/or LoadOrchestratorRoutes()
// 3. Start server with Serve()
// 4. Gracefully shutdown with Shutdown()
type TKAServer struct {
	// Options
	debug bool

	// API
	router *gin.Engine
	tracer trace.Tracer

	// Auth service
	client         client.TkaClient
	authMiddleware mw.Middleware

	// API behavior
	retryAfterSeconds int

	// Tailnet Server
	tsServer *ts.Server
}

// NewTKAServer creates a new TKAServer instance with the provided Tailscale server and options.
// This is the primary constructor for the TKA HTTP API server.
//
// Parameters:
//   - srv: A configured tailscale.Server that handles network connectivity and TLS
//   - _: Reserved parameter for future use (pass nil)
//   - opts: Zero or more Option functions to customize server behavior
//
// Returns:
//   - *TKAServer: Configured server ready for route loading and serving
//   - humane.Error: Error if server creation fails
//
// The constructor automatically:
//   - Sets up Gin router with observability middleware (tracing, logging, metrics)
//   - Configures default Tailscale authentication middleware
//   - Establishes Swagger documentation endpoint
//   - Applies all provided options
//
// Example:
//
//	server, err := NewTKAServer(tailscaleServer, nil,
//	  WithDebug(true),
//	  WithRetryAfterSeconds(5),
//	)
//	if err != nil {
//	  return err
//	}
//
// Note: You must call LoadApiRoutes() and/or LoadOrchestratorRoutes() before serving.
func NewTKAServer(srv *ts.Server, _ any, opts ...Option) (*TKAServer, humane.Error) {
	capName := tailcfg.PeerCapability(viper.GetString("tailscale.capName"))
	defaultAuthMiddleware := authMw.NewGinAuthMiddleware[capability.Rule](srv, capName)

	tkaServer := &TKAServer{
		debug:             false,
		router:            nil,
		tracer:            otel.Tracer("tka"),
		client:            nil,
		authMiddleware:    defaultAuthMiddleware,
		retryAfterSeconds: 1,
		tsServer:          srv,
	}

	// Apply Options
	for _, opt := range opts {
		opt(tkaServer)
	}

	// Setup Gin router
	tkaServer.router = utils.NewO11yGin("tka_server", tkaServer.debug)

	tkaServer.loadStaticRoutes()
	return tkaServer, nil
}

func (t *TKAServer) loadStaticRoutes() {
	// serve K8s controller metrics on /metrics/controller
	t.router.GET("/metrics/controller", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	// Add Swagger documentation endpoint
	// This will serve the Swagger UI at /swagger/index.html
	t.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Optionally, add a redirect from /swagger to /swagger/index.html
	t.router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(301, "/swagger/index.html")
	})
}

// LoadApiRoutes registers the authentication API endpoints with the server.
// This method must be called before Serve() to enable user authentication functionality.
//
// Parameters:
//   - svc: Service implementation for handling authentication business logic
//
// Returns:
//   - humane.Error: Error if service is nil or route registration fails
//
// Registered endpoints:
//   - POST /api/v1alpha1/login - Authenticate user and provision credentials
//   - GET /api/v1alpha1/login - Check current authentication status
//   - GET /api/v1alpha1/kubeconfig - Retrieve kubeconfig for authenticated user
//   - POST /api/v1alpha1/logout - Revoke user credentials
//
// Example:
//
//	authService := service.NewOperatorService(operatorOpts)
//	if err := server.LoadApiRoutes(authService); err != nil {
//	  return err
//	}
func (t *TKAServer) LoadApiRoutes(svc client.TkaClient) humane.Error {
	if svc == nil {
		return humane.New("auth service not configured", "Provide a k8s.TkaClient via api.WithAuthService option")
	}
	t.client = svc

	v1alpha1Grpup := t.router.Group(ApiRouteV1Alpha1)

	// Install auth middleware only on the API route group
	if t.authMiddleware != nil {
		t.authMiddleware.UseGroup(v1alpha1Grpup, t.tracer)
	}

	v1alpha1Grpup.POST(LoginApiRoute, t.login)
	v1alpha1Grpup.GET(LoginApiRoute, t.getLogin)
	v1alpha1Grpup.GET(KubeconfigApiRoute, t.getKubeconfig)
	v1alpha1Grpup.POST(LogoutApiRoute, t.logout)

	return nil
}

// Serve starts the TKA server with TLS setup and HTTP functionality, handling Tailnet connection and request serving.
// It listens on the configured port and returns wrapped errors for any issues encountered during initialization or runtime.
func (t *TKAServer) Serve(ctx context.Context) humane.Error {
	if t.tsServer == nil {
		return humane.New("tailscale server not configured", "Provide a tailscale.Server via api.WithTailnetServer option")
	}
	return t.tsServer.Serve(ctx, t.router)
}

// Shutdown gracefully stops the tka server if it is running, releasing any resources and handling in-progress requests.
// It returns a humane.Error if the server fails to stop.
func (t *TKAServer) Shutdown(ctx context.Context) humane.Error {
	if t.tsServer == nil {
		return nil
	}
	return t.tsServer.Shutdown(ctx)
}

// Engine returns the underlying gin.Engine for advanced integration scenarios.
// This method is primarily intended for testing and advanced embedding use cases
// where direct access to the Gin router is required.
//
// Returns:
//   - *gin.Engine: The underlying Gin router instance
//
// Use cases:
//   - Adding custom middleware in tests
//   - Embedding the server in larger applications
//   - Advanced route inspection and modification
//
// Example:
//
//	engine := server.Engine()
//	engine.Use(customTestMiddleware())
func (t *TKAServer) Engine() *gin.Engine { return t.router }

// Use attaches middleware to the underlying Gin router.
// This method allows external packages and tests to add custom middleware
// to the server's request processing pipeline.
//
// Parameters:
//   - mw: One or more Gin middleware handler functions
//
// The middleware will be applied to all routes registered after this call.
// For route-specific middleware, access the router via Engine() method.
//
// Example:
//
//	server.Use(gin.Logger())
//	server.Use(customAuthMiddleware())
func (t *TKAServer) Use(mw ...gin.HandlerFunc) { t.router.Use(mw...) }
