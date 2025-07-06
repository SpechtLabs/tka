package main

import (
	"fmt"
	"os"

	//"github.com/spechtlabs/go-otel-utils/otelprovider"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"
)

var debug bool

func main() {
	//logProvider := otelprovider.NewLogger(
	//	otelprovider.WithLogAutomaticEnv(),
	//)
	//
	//traceProvider := otelprovider.NewTracer(
	//	otelprovider.WithTraceAutomaticEnv(),
	//)

	// Initialize Logging
	debug = os.Getenv("OTEL_LOG_LEVEL") == "debug"
	var zapLogger *zap.Logger
	var err error
	if debug {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
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
		//otelzap.WithLoggerProvider(logProvider),
	)

	// Replace global otelZap logger
	undoOtelZapGlobals := otelzap.ReplaceGlobals(otelZapLogger)

	defer func() {
		//if err := traceProvider.ForceFlush(context.Background()); err != nil {
		//	otelzap.L().Warn("failed to flush traces")
		//}
		//
		//if err := logProvider.ForceFlush(context.Background()); err != nil {
		//	otelzap.L().Warn("failed to flush logs")
		//}
		//
		//if err := traceProvider.Shutdown(context.Background()); err != nil {
		//	panic(err)
		//}
		//
		//if err := logProvider.Shutdown(context.Background()); err != nil {
		//	panic(err)
		//}

		undoStdLogRedirect()
		undoOtelZapGlobals()
		undoZapGlobals()
	}()

	Execute()
}
