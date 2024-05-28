package graceful

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go.uber.org/zap"
)

// NewContext constructs a new graceful context. The returned contexts embeds
// the context.Context interface and exposes additional methods which could be
// used to gracefully shutdown components and connections before exiting the application.
func NewContext(bg context.Context) Context {
	bg, cancel := context.WithCancel(bg)

	return &ctx{
		Context: bg,
		cancel:  cancel,
		sigChan: make(chan os.Signal, 1), // Separate signal channel for each context
	}
}

type Context interface {
	context.Context
	// AwaitKillSignal blocks until a kill signal is received.
	AwaitKillSignal()
	// Shutdown cancels the context and waits for all registered closers to finish.
	// The application is force quit if a second sigterm or sigquit is received.
	Shutdown()
	// Closer registers a function to be called when the context is cancelled.
	// Closer is a blocking function.
	Closer(fn func())
}

type ctx struct {
	context.Context
	mu      sync.Mutex
	wg      sync.WaitGroup
	cancel  context.CancelFunc
	logger  *zap.Logger
	sigChan chan os.Signal // Separate signal channel for each context
}

// goroutine is spawned to wait for all registered closers to finish their operations.
// blocking on (ctx.wg) by incrementing for each closer till the closer finished closing.
func (ctx *ctx) Closer(fn func()) {
	ctx.mu.Lock()
	ctx.wg.Add(1)
	ctx.mu.Unlock()
	defer ctx.wg.Done()
	<-ctx.Done()
	fn()
}

// cancel triggers Done channel to close, releasing any goroutines/operations waiting on channel.
func (ctx *ctx) Shutdown() {
	ctx.cancel()
	fmt.Println("graceful shutdown, press Ctrl_C to force quit")

	closer := make(chan struct{})
	go func() {
		defer close(closer)
		ctx.mu.Lock()
		ctx.wg.Wait()
		ctx.mu.Unlock()
	}()

	// waits on either the context's specific signal channel or the closer channel.
	// If a signal is received it either force quits
	// or, if all closers have finished, it exits gracefully.
	force := make(chan os.Signal, 1)
	signal.Notify(force, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-force:
		fmt.Println("force quitting")
	case <-ctx.sigChan: // Wait for the specific context's signal channel to exit
		if ctx.logger != nil {
			fmt.Println("signal quitting")
		}
	case <-closer:
	}
}

// AwaitKillSignal blocks current process till interrupt signal is received
// and the context has closed
func (ctx *ctx) AwaitKillSignal() {
	fmt.Println("awaiting kill signal")

	exit, done := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer done()

	<-exit.Done()

	fmt.Println("kill signal received")
	ctx.Shutdown()
}
