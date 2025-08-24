package api

import (
	"context"
	"time"

	// gin
	"github.com/gin-gonic/gin"
	_ "github.com/spechtlabs/tailscale-k8s-auth/pkg/swagger"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	// Misc
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"

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

	// Tailscale
	"tailscale.com/tailcfg"
)

// @title Tailscale Kubernetes Auth API
// @version 1.0
// @description API for authenticating and authorizing Kubernetes access via Tailscale identity.
// @contact.name Specht Labs
// @contact.url specht-labs.de
// @contact.email tka@specht-labs.de
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host tka.sphinx-map.ts.net:8123
// @BasePath /api/v1alpha1
// @securityDefinitions.apikey TailscaleAuth
// @in header
// @name X-Tailscale-User
// @description Authentication happens automatically via the Tailscale network. The server performs a WhoIs lookup on the client's IP address to determine identity. This header is for documentation purposes only and is not actually required to be set.
const (
	ApiRouteV1Alpha1   = "/api/v1alpha1"
	LoginApiRoute      = "/login"
	KubeconfigApiRoute = "/kubeconfig"
	LogoutApiRoute     = "/logout"
)

type TKAServer struct {
	// Options
	debug   bool
	capName tailcfg.PeerCapability

	// Tailscale Server
	tsServer *tailscale.Server

	// API
	router *gin.Engine
	tracer trace.Tracer

	// Kuberneters Operator
	operator *operator.KubeOperator

	// API behavior
	retryAfterSeconds int
}

func NewTKAServer(srv *tailscale.Server, operator *operator.KubeOperator, opts ...Option) (*TKAServer, humane.Error) {
	tkaServer := &TKAServer{
		debug:             false,
		capName:           "specht-labs.de/cap/tka",
		tsServer:          srv,
		router:            nil,
		tracer:            otel.Tracer("tka"),
		operator:          operator,
		retryAfterSeconds: 1,
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

	authMiddleware := tailscale.NewGinAuthMiddlewareFromServer[capRule](tkaServer.tsServer, tkaServer.capName)
	authMiddleware.Use(tkaServer.router, tkaServer.tracer)

	// Set-up routes
	v1alpha1Grpup := tkaServer.router.Group(ApiRouteV1Alpha1)
	v1alpha1Grpup.POST(LoginApiRoute, tkaServer.login)
	v1alpha1Grpup.GET(LoginApiRoute, tkaServer.getLogin)
	v1alpha1Grpup.GET(KubeconfigApiRoute, tkaServer.getKubeconfig)
	v1alpha1Grpup.POST(LogoutApiRoute, tkaServer.logout)

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

// Serve starts the TKA server with TLS setup and HTTP functionality, handling Tailnet connection and request serving.
// It listens on the configured port and returns wrapped errors for any issues encountered during initialization or runtime.
func (t *TKAServer) Serve(ctx context.Context) humane.Error {
	return t.tsServer.Serve(ctx, t.router)
}

// Shutdown gracefully stops the tka server if it is running, releasing any resources and handling in-progress requests.
// It returns a humane.Error if the server fails to stop.
func (t *TKAServer) Shutdown(ctx context.Context) humane.Error {
	return t.tsServer.Shutdown(ctx)
}
