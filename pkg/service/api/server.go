package api

import (
	"net/http"

	// gin
	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/tka/internal/utils"
	client "github.com/spechtlabs/tka/pkg/client/k8s"
	"github.com/spechtlabs/tka/pkg/cluster"
	mw "github.com/spechtlabs/tka/pkg/middleware"
	"github.com/spechtlabs/tka/pkg/service"
	"github.com/spechtlabs/tka/pkg/service/models"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	ginprometheus "github.com/zsais/go-gin-prometheus"

	// Misc
	"github.com/sierrasoftworks/humane-errors-go"

	// Logging

	// o11y
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	// tka
)

// API route constants define the URL paths for the TKA REST API.
const (
	// ApiRouteV1Alpha1 is the base path for the v1alpha1 API version.
	ApiRouteV1Alpha1 = "/api/v1alpha1"
	// LoginApiRoute is the path for user login operations.
	LoginApiRoute = "/login"
	// KubeconfigApiRoute is the path for retrieving kubeconfig files.
	KubeconfigApiRoute = "/kubeconfig"
	// LogoutApiRoute is the path for user logout operations.
	LogoutApiRoute = "/logout"
	// ClusterInfoApiRoute is the path for retrieving cluster information.
	ClusterInfoApiRoute = "/cluster-info"
	MemberlistRoute     = "/memberlist"
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
	// API
	router           *gin.Engine
	tracer           trace.Tracer
	sharedPrometheus *ginprometheus.Prometheus
	clusterInfo      *models.TkaClusterInfo

	// Auth service
	client         client.TkaClient
	authMiddleware mw.Middleware

	// Gossip
	gossipStore cluster.GossipStore[service.NodeMetadata]

	// API behavior
	retryAfterSeconds int
}

// NewTKAServer creates a new TKAServer instance with the provided options.
// This is the primary constructor for the TKA HTTP API server.
//
// The constructor automatically:
//   - Sets up Gin router with observability middleware (tracing, logging, metrics)
//   - Configures default Tailscale authentication middleware
//   - Establishes Swagger documentation endpoint
//   - Applies all provided options
//
// It returns the configured server or an error if initialization fails.
//
// Example:
//
//	server, err := NewTKAServer(
//	  WithRetryAfterSeconds(5),
//	)
//	if err != nil {
//	  return err
//	}
//
// Note: You must call LoadApiRoutes() and/or LoadOrchestratorRoutes() before serving.
func NewTKAServer(opts ...Option) *TKAServer {
	tkaServer := &TKAServer{
		router:            nil,
		tracer:            otel.Tracer("tka"),
		client:            nil,
		authMiddleware:    nil,
		retryAfterSeconds: 1,
		sharedPrometheus:  nil,
		clusterInfo:       nil,
		gossipStore:       nil,
	}

	// Apply Options
	for _, opt := range opts {
		opt(tkaServer)
	}

	if tkaServer.sharedPrometheus == nil {
		tkaServer.sharedPrometheus = ginprometheus.NewPrometheus("tka_server")
	}

	tkaServer.router = utils.NewO11yGin("tka_server", tkaServer.sharedPrometheus)

	tkaServer.loadTemplates()
	tkaServer.loadStaticRoutes()
	return tkaServer
}

// loadStaticRoutes registers static endpoints and the Swagger UI.
func (t *TKAServer) loadStaticRoutes() {
	// Add Swagger documentation endpoint
	// This will serve the Swagger UI at /swagger/index.html
	t.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Optionally, add a redirect from /swagger to /swagger/index.html
	t.router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	if t.gossipStore != nil {
		t.router.GET(MemberlistRoute, t.getMemberlist)
	}
}

// LoadApiRoutes registers the authentication API endpoints with the server.
// It must be called before Serve() to enable user authentication functionality.
// It returns an error if the provided service implementation (svc) is nil.
//
// The following endpoints are registered:
//   - POST /api/v1alpha1/login        - Authenticate user and provision credentials
//   - GET  /api/v1alpha1/login        - Check current authentication status
//   - GET  /api/v1alpha1/kubeconfig   - Retrieve kubeconfig for authenticated user
//   - POST /api/v1alpha1/logout       - Revoke user credentials
//   - GET  /api/v1alpha1/cluster-info - Retrieve cluster information
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
	v1alpha1Grpup.GET(ClusterInfoApiRoute, t.getClusterInfo)

	return nil
}

// Engine returns the underlying gin.Engine for advanced integration scenarios.
// This method is primarily intended for testing and advanced embedding use cases
// where direct access to the Gin router is required.
func (t *TKAServer) Engine() *gin.Engine { return t.router }
