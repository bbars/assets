package ctxutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func HandleInterruptSignal(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-interrupt
		cancel()
	}()

	return ctx
}
