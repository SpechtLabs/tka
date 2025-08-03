package api

import (
	"context"
	"time"

	// misc
	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	server2 "github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"

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

const (
	LoginApiRoute      = "/login"
	KubeconfigApiRoute = "/kubeconfig"
	LogoutApiRoute     = "/logout"
)

type TKAServer struct {
	// Options
	debug   bool
	capName tailcfg.PeerCapability

	// Tailscale Server
	tsServer *server2.Server

	// API
	router *gin.Engine
	tracer trace.Tracer

	// Kuberneters Operator
	operator *operator.KubeOperator
}

func NewTKAServer(srv *server2.Server, operator *operator.KubeOperator, opts ...Option) (*TKAServer, humane.Error) {
	tkaServer := &TKAServer{
		debug:    false,
		capName:  "specht-labs.de/cap/tka",
		tsServer: srv,
		router:   nil,
		tracer:   otel.Tracer("tka"),
		operator: operator,
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
	tkaServer.router = gin.New(func(e *gin.Engine) {})
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

	// Set-up routes
	tkaServer.router.POST(KubeconfigApiRoute, tkaServer.login)
	tkaServer.router.GET(LoginApiRoute, tkaServer.getLogin)
	tkaServer.router.POST(LoginApiRoute, tkaServer.login)

	tkaServer.router.GET(KubeconfigApiRoute, tkaServer.getKubeconfig)

	tkaServer.router.DELETE(KubeconfigApiRoute, tkaServer.logout)
	tkaServer.router.DELETE(LoginApiRoute, tkaServer.logout)
	tkaServer.router.DELETE(LogoutApiRoute, tkaServer.logout)
	tkaServer.router.GET(LogoutApiRoute, tkaServer.logout)

	// serve K8s controller metrics on /metrics/controller
	tkaServer.router.GET("/metrics/controller", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	return tkaServer, nil
}

// Serve starts the TKA tailscale with TLS setup and HTTP tailscale functionality, handling Tailnet connection and request serving.
// It listens on the configured port and returns wrapped errors for any issues encountered during initialization or runtime.
func (t *TKAServer) Serve(ctx context.Context) humane.Error {
	return t.tsServer.Serve(ctx, t.router)
}

// Shutdown gracefully stops the tka tailscale if it is running, releasing any resources and handling in-progress requests.
// It returns a humane.Error if the tailscale fails to stop.
func (t *TKAServer) Shutdown(ctx context.Context) humane.Error {
	return t.tsServer.Shutdown(ctx)
}
