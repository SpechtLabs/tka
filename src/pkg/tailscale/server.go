package tailscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	// misc
	"github.com/gin-gonic/gin"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"

	// Logging
	ginzap "github.com/gin-contrib/zap"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	// Tailscale
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

type TKAServer struct {
	// Options
	debug    bool
	port     int
	stateDir string
	capName  tailcfg.PeerCapability

	// Tailscale Server
	ts        *tsnet.Server
	lc        *local.Client
	st        *ipnstate.Status
	serverURL string // "https://foo.bar.ts.net"

	// API
	srv    *http.Server
	router *gin.Engine
	tracer trace.Tracer

	// Kuberneters Operator
	operator *operator.KubeOperator
}

func NewTKAServer(ctx context.Context, hostname string, operator *operator.KubeOperator, opts ...Option) (*TKAServer, humane.Error) {
	tkaServer := &TKAServer{
		debug:     false,
		port:      443,
		stateDir:  "",
		capName:   "specht-labs.de/cap/tka",
		ts:        nil,
		lc:        nil,
		st:        nil,
		serverURL: "",
		srv:       nil,
		router:    nil,
		tracer:    otel.Tracer("tka"),
		operator:  operator,
	}

	// Apply Options
	for _, opt := range opts {
		opt(tkaServer)
	}

	// Connect to the tailnet
	tkaServer.ts = &tsnet.Server{
		Hostname: hostname,
		Dir:      tkaServer.stateDir,
	}

	if tkaServer.debug {
		tkaServer.ts.Logf = otelzap.L().Sugar().Debugf
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
	tkaServer.router.POST("/kubeconfig", tkaServer.login)
	tkaServer.router.GET("/login", tkaServer.login)
	tkaServer.router.POST("/login", tkaServer.login)

	tkaServer.router.GET("/kubeconfig", tkaServer.getKubeconfig)

	tkaServer.router.DELETE("/kubeconfig", tkaServer.logout)
	tkaServer.router.DELETE("/login", tkaServer.logout)
	tkaServer.router.DELETE("/logout", tkaServer.logout)
	tkaServer.router.GET("/logout", tkaServer.logout)

	// serve K8s controller metrics on /metrics/controller
	tkaServer.router.GET("/metrics/controller", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	return tkaServer, nil
}

// Serve starts the TKA server with TLS setup and HTTP server functionality, handling Tailnet connection and request serving.
// It listens on the configured port and returns wrapped errors for any issues encountered during initialization or runtime.
func (t *TKAServer) Serve(ctx context.Context) humane.Error {
	otelzap.L().Info("Starting TKA Server", zap.String("address", t.serverURL))

	if err := t.connectTailnet(ctx); err != nil {
		return humane.Wrap(err, "failed to connect to tailnet", "check (debug) logs for more details")
	}

	//listener, err := t.ts.ListenTLS("tcp", fmt.Sprintf(":%d", t.port))
	listener, err := t.ts.Listen("tcp", fmt.Sprintf(":%d", t.port))
	if err != nil {
		return humane.Wrap(err, "failed to listen on port",
			"check (debug) logs for more details",
			"check that port is not in use. if you are using a port that is in use, you can use the --port flag to specify a different port",
			"if you use privileged ports, you may need to run as root",
		)
	}

	// configure the HTTP Server
	t.srv = &http.Server{
		Handler: t.router,
	}

	err = t.srv.Serve(listener)
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			otelzap.L().Info("tka server stopped")
			return nil
		} else {
			otelzap.L().WithError(err).ErrorContext(ctx, "Failed to start TKA server")
		}
	}

	return nil
}

// Shutdown gracefully stops the tka server if it is running, releasing any resources and handling in-progress requests.
// It returns a humane.Error if the server fails to stop.
func (t *TKAServer) Shutdown() humane.Error {
	if t.srv == nil {
		return humane.New("Unable to shutdown tka Server. It is not running.", "Make sure the tka server is running and try again.")
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	otelzap.L().Info("shutting down tka")
	if err := t.srv.Shutdown(ctx); err != nil {
		return humane.Wrap(err, "Unable to shutdown tka server", "Make sure the tka server is running and try again.")
	}

	return nil
}

func (t *TKAServer) connectTailnet(ctx context.Context) humane.Error {
	var err error
	t.st, err = t.ts.Up(ctx)
	if err != nil {
		return humane.Wrap(err, "failed to start tailscale server", "check (debug) logs for more details")
	}

	portSuffix := ""
	if t.port != 443 {
		portSuffix = fmt.Sprintf(":%d", t.port)
	}

	t.serverURL = fmt.Sprintf("https://%s%s", strings.TrimSuffix(t.st.Self.DNSName, "."), portSuffix)

	if t.lc, err = t.ts.LocalClient(); err != nil {
		return humane.Wrap(err, "failed to get local tailscale client", "check (debug) logs for more details")
	}

	otelzap.L().InfoContext(ctx, "tka server running", zap.String("url", t.serverURL))
	return nil
}
