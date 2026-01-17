package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spechtlabs/go-otel-utils/otelprovider"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// InitObservability initializes observability tools including logging and tracing, and returns a cleanup function.
func InitObservability() func() {
	var loggerOptions []otelprovider.LoggerOption
	var tracerOptions []otelprovider.TracerOption

	otelEndpoint := viper.GetString("otel.endpoint")

	if otelInsecure := viper.GetBool("otel.insecure"); otelInsecure {
		loggerOptions = append(loggerOptions, otelprovider.WithLogInsecure())
		tracerOptions = append(tracerOptions, otelprovider.WithTraceInsecure())
	}

	if strings.Contains(otelEndpoint, "4317") {
		loggerOptions = append(loggerOptions, otelprovider.WithGrpcLogEndpoint(otelEndpoint))
		tracerOptions = append(tracerOptions, otelprovider.WithGrpcTraceEndpoint(otelEndpoint))
	} else if strings.Contains(otelEndpoint, "4318") {
		loggerOptions = append(loggerOptions, otelprovider.WithHttpLogEndpoint(otelEndpoint))
		tracerOptions = append(tracerOptions, otelprovider.WithHttpTraceEndpoint(otelEndpoint))
	}

	logProvider := otelprovider.NewLogger(loggerOptions...)
	traceProvider := otelprovider.NewTracer(tracerOptions...)

	// Initialize Logging
	debug := viper.GetBool("debug")
	var zapLogger *zap.Logger
	var err error
	if debug {
		zapLogger, err = zap.NewDevelopment()
		gin.SetMode(gin.DebugMode)
	} else {
		zapLogger, err = zap.NewProduction()
		gin.SetMode(gin.ReleaseMode)
	}
	if err != nil {
		fmt.Printf("failed to initialize logger: %v", err) //nolint:golint-sl // Pre-logger init output
		os.Exit(1)
	}

	// Replace zap global
	undoZapGlobals := zap.ReplaceGlobals(zapLogger)

	// Redirect stdlib log to zap
	undoStdLogRedirect := zap.RedirectStdLog(zapLogger)

	// Create otelLogger
	otelZapLogger := otelzap.New(zapLogger,
		otelzap.WithCaller(true),
		otelzap.WithMinLevel(zap.InfoLevel),
		otelzap.WithAnnotateLevel(zap.WarnLevel),
		otelzap.WithErrorStatusLevel(zap.ErrorLevel),
		otelzap.WithStackTrace(false),
		otelzap.WithLoggerProvider(logProvider),
	)

	// Replace global otelZap logger
	undoOtelZapGlobals := otelzap.ReplaceGlobals(otelZapLogger)

	return func() {
		// Capture errors for wide event
		var (
			traceFlushErr    error
			logFlushErr      error
			traceShutdownErr error
			logShutdownErr   error
		)

		traceFlushErr = traceProvider.ForceFlush(context.Background())
		logFlushErr = logProvider.ForceFlush(context.Background())
		traceShutdownErr = traceProvider.Shutdown(context.Background())
		logShutdownErr = logProvider.Shutdown(context.Background())

		// Emit single wide event for observability shutdown with all error details
		otelzap.L().Info("observability shutdown",
			zap.Bool("trace_flush_ok", traceFlushErr == nil),
			zap.Bool("log_flush_ok", logFlushErr == nil),
			zap.Bool("trace_shutdown_ok", traceShutdownErr == nil),
			zap.Bool("log_shutdown_ok", logShutdownErr == nil),
			zap.NamedError("trace_flush_err", traceFlushErr),
			zap.NamedError("log_flush_err", logFlushErr),
			zap.NamedError("trace_shutdown_err", traceShutdownErr),
			zap.NamedError("log_shutdown_err", logShutdownErr),
		)

		undoStdLogRedirect()
		undoOtelZapGlobals()
		undoZapGlobals()
	}
}
