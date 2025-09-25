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
	otelInsecure := viper.GetBool("otel.insecure")

	if otelInsecure {
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
		fmt.Printf("failed to initialize logger: %v", err)
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
		if err := traceProvider.ForceFlush(context.Background()); err != nil {
			otelzap.L().Warn("failed to flush traces")
		}

		if err := logProvider.ForceFlush(context.Background()); err != nil {
			otelzap.L().Warn("failed to flush logs")
		}

		if err := traceProvider.Shutdown(context.Background()); err != nil {
			panic(err)
		}

		if err := logProvider.Shutdown(context.Background()); err != nil {
			panic(err)
		}

		undoStdLogRedirect()
		undoOtelZapGlobals()
		undoZapGlobals()
	}
}
