package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"go.uber.org/zap"
)

func InterruptHandler(ctx context.Context, cancelCtx context.CancelCauseFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		defer signal.Stop(sigs) // Clean up signal notifications

		select {
		// Wait for context cancel
		case <-ctx.Done():
			return

		// Wait for signal
		case sig := <-sigs:
			switch sig {
			case syscall.SIGTERM:
				otelzap.L().Debug("Received SIGTERM, initiating graceful shutdown...")
				cancelCtx(context.Canceled)
			case syscall.SIGINT:
				otelzap.L().Debug("Received SIGINT (Ctrl+C), initiating graceful shutdown...")
				cancelCtx(context.Canceled)
			case syscall.SIGQUIT:
				otelzap.L().Debug("Received SIGQUIT, initiating graceful shutdown...")
				cancelCtx(context.Canceled)
			default:
				otelzap.L().WarnContext(ctx, "Received unknown signal", zap.String("signal", sig.String()))
			}
		}
	}()
}
