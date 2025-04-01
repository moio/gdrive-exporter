package utils

import (
	"context"
	"os"
	"os/signal"
)

func NewSignalCancelingContext() (context.Context, context.CancelFunc) {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		select {
		case <-signalChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, func() {
		signal.Stop(signalChan)
		cancel()
	}
}
