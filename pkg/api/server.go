package api

import (
	"context"
	"time"

	// gin
	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/tka/pkg/auth"
	mw "github.com/spechtlabs/tka/pkg/middleware/auth"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Misc
	"github.com/sierrasoftworks/humane-errors-go"

	// Logging
	ginzap "github.com/gin-contrib/zap"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// o11y
	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	// tka
	ts "github.com/spechtlabs/tka/pkg/tailscale"
)

// @title Tailscale Kubernetes Auth API
// @version 1.0
// @description API for authenticating and authorizing Kubernetes access via Tailscale identity.
// @contact.name Specht Labs
// @contact.url specht-labs.de
// @contact.email tka@specht-labs.de
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /api/v1alpha1
// @securityDefinitions.apikey TailscaleAuth
// @in header
// @name X-Tailscale-User
// @description Authentication happens automatically via the Tailscale network. The server performs a WhoIs lookup on the client's IP address to determine identity. This header is for documentation purposes only and is not actually required to be set.
const (
	ApiRouteV1Alpha1          = "/api/v1alpha1"
	OrchestratorRouteV1Alpha1 = "/orchestrator/v1alpha1"
	LoginApiRoute             = "/login"
	KubeconfigApiRoute        = "/kubeconfig"
	LogoutApiRoute            = "/logout"
	ClustersRoute             = "/clusters"
)

type TKAServer struct {
	// Options
	debug bool

	// API
	router *gin.Engine
	tracer trace.Tracer

	// Auth service
	auth   auth.Service
	authMW mw.Middleware

	// API behavior
	retryAfterSeconds int

	// Tailnet Server
	tsServer *ts.Server
}

func NewTKAServer(_ any, _ any, opts ...Option) (*TKAServer, humane.Error) {
	tkaServer := &TKAServer{
		debug:             false,
		router:            nil,
		tracer:            otel.Tracer("tka"),
		auth:              nil,
		authMW:            nil,
		retryAfterSeconds: 1,
		tsServer:          nil,
	}

	// Apply Options
	for _, opt := range opts {
		opt(tkaServer)
	}

	if tkaServer.debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup Gin router
	tkaServer.router = gin.New()
	tkaServer.router.Use(otelgin.Middleware("tka_server"))
	// Setup ginzap to log everything correctly to zap
	tkaServer.router.Use(ginzap.GinzapWithConfig(otelzap.L(), &ginzap.Config{
		UTC:        true,
		TimeFormat: time.RFC3339,
		Context: func(c *gin.Context) []zapcore.Field {
			var fields []zapcore.Field
			// log request ID
			if requestID := c.Writer.Header().Get("X-Request-Id"); requestID != "" {
				fields = append(fields, zap.String("request_id", requestID))
			}

			// log trace and span ID
			if trace.SpanFromContext(c.Request.Context()).SpanContext().IsValid() {
				fields = append(fields, zap.String("trace_id", trace.SpanFromContext(c.Request.Context()).SpanContext().TraceID().String()))
				fields = append(fields, zap.String("span_id", trace.SpanFromContext(c.Request.Context()).SpanContext().SpanID().String()))
			}
			return fields
		},
	}))

	// Set-up Prometheus to expose prometheus metrics
	p := ginprometheus.NewPrometheus("tka_server")
	p.Use(tkaServer.router)

	// Install injected auth middleware if provided
	if tkaServer.authMW != nil {
		tkaServer.authMW.Use(tkaServer.router, tkaServer.tracer)
	}

	// serve K8s controller metrics on /metrics/controller
	tkaServer.router.GET("/metrics/controller", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	// Add Swagger documentation endpoint
	// This will serve the Swagger UI at /swagger/index.html
	tkaServer.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Optionally, add a redirect from /swagger to /swagger/index.html
	tkaServer.router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(301, "/swagger/index.html")
	})

	return tkaServer, nil
}

func (t *TKAServer) LoadApiRoutes() {
	v1alpha1Grpup := t.router.Group(ApiRouteV1Alpha1)
	v1alpha1Grpup.POST(LoginApiRoute, t.login)
	v1alpha1Grpup.GET(LoginApiRoute, t.getLogin)
	v1alpha1Grpup.GET(KubeconfigApiRoute, t.getKubeconfig)
	v1alpha1Grpup.POST(LogoutApiRoute, t.logout)
}

func (t *TKAServer) LoadOrchestratorRoutes() {
	v1alpha1Grpup := t.router.Group(OrchestratorRouteV1Alpha1)
	v1alpha1Grpup.GET(ClustersRoute, t.getClusters)
	v1alpha1Grpup.POST(ClustersRoute, t.registerCluster)
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

// Engine returns the underlying gin.Engine to facilitate external package tests and advanced embedding.
func (t *TKAServer) Engine() *gin.Engine { return t.router }

// Use allows attaching middleware to the underlying router from external packages/tests.
func (t *TKAServer) Use(mw ...gin.HandlerFunc) { t.router.Use(mw...) }
