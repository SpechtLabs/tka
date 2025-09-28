package api

import (

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
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"tailscale.com/tailcfg"

	// Misc
	"github.com/sierrasoftworks/humane-errors-go"

	// Logging

	// o11y
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	// tka
	ts "github.com/spechtlabs/tka/pkg/tshttp"
)

const (
	OrchestratorRouteV1Alpha1 = "/orchestrator/v1alpha1"
	ClustersRoute             = "/clusters"
)

type TKAServer struct {
	// API
	router           *gin.Engine
	tracer           trace.Tracer
	sharedPrometheus *ginprometheus.Prometheus

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
//	  WithRetryAfterSeconds(5),
//	)
//	if err != nil {
//	  return err
//	}
//
// Note: You must call LoadApiRoutes() and/or LoadOrchestratorRoutes() before serving.
func NewTKAServer(srv *ts.Server, opts ...Option) (*TKAServer, humane.Error) {
	capName := tailcfg.PeerCapability(viper.GetString("tailscale.capName"))
	defaultAuthMiddleware := authMw.NewGinAuthMiddleware[capability.Rule](srv, capName)

	tkaServer := &TKAServer{
		router:            nil,
		tracer:            otel.Tracer("tka"),
		client:            nil,
		authMiddleware:    defaultAuthMiddleware,
		retryAfterSeconds: 1,
		tsServer:          srv,
		sharedPrometheus:  nil,
	}

	// Apply Options
	for _, opt := range opts {
		opt(tkaServer)
	}

	if tkaServer.sharedPrometheus == nil {
		tkaServer.sharedPrometheus = ginprometheus.NewPrometheus("tka_orchestrator")
	}

	// Setup Gin router
	tkaServer.router = utils.NewO11yGin("tka_orchestrator", tkaServer.sharedPrometheus)

	tkaServer.loadStaticRoutes()
	return tkaServer, nil
}

func (t *TKAServer) loadStaticRoutes() {
	// Add Swagger documentation endpoint
	// This will serve the Swagger UI at /swagger/index.html
	t.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Optionally, add a redirect from /swagger to /swagger/index.html
	t.router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(301, "/swagger/index.html")
	})
}

// LoadOrchestratorRoutes registers the cluster orchestration API endpoints with the server.
// These routes are used for multi-cluster management and service discovery.
//
// Returns:
//   - humane.Error: Error if route registration fails
//
// Registered endpoints:
//   - GET /orchestrator/v1alpha1/clusters - List all available clusters
//   - POST /orchestrator/v1alpha1/clusters - Register a new cluster
//
// Note: These routes are optional and only needed for multi-cluster deployments.
//
// Example:
//
//	if err := server.LoadOrchestratorRoutes(); err != nil {
//	  return err
//	}
func (t *TKAServer) LoadOrchestratorRoutes() humane.Error {
	v1alpha1Grpup := t.router.Group(OrchestratorRouteV1Alpha1)

	// Install auth middleware only on the orchestrator route group
	if t.authMiddleware != nil {
		t.authMiddleware.UseGroup(v1alpha1Grpup, t.tracer)
	}

	v1alpha1Grpup.GET(ClustersRoute, t.getClusters)
	v1alpha1Grpup.POST(ClustersRoute, t.registerCluster)
	return nil
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
