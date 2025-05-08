package shutdown

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rohanthewiz/logger"
)

const gracePeriod = 15 * time.Second

type HookFunc func(duration time.Duration) error

type shutdownHooks struct {
	Hooks []HookFunc
	lock  sync.Mutex
}

var hooks shutdownHooks

func RegisterHook(fn HookFunc) {
	hooks.lock.Lock()
	defer hooks.lock.Unlock()
	hooks.Hooks = append(hooks.Hooks, fn)
	fmt.Printf("Registered shutdown hook: #%d\n", len(hooks.Hooks))
}

// InitShutdownService initializes the shutdown service, so things can shutdown gracefully
// It will close the done channel to allow the app to shutdown
func InitShutdownService(done chan struct{}) {
	// Setup shutdown signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Go handle shutdown signal
	go func() {
		defer close(done)
		wg := sync.WaitGroup{}

		sig := <-sigChan
		log.Printf("Received shutdown signal: %v", sig)
		setShutdown()

		/* Hmmm, maybe we do want to kill with a second CTRL-C.
		// Keep capturing signals so that subsequent CTRL-C's
		// 	don't kill us by default.
		go func() {
			// we're going to consume ALL future SIGINTs so they
			// don't fall through to the kernel's default.
			for sig := range sigChan {
				log.Printf("caught subsequent signal: %v", sig)

			}
		}() */

		// Fire all shutdown hooks
		log.Printf("Shutting down %d hooks (grace period is: %s)", len(hooks.Hooks), gracePeriod)

		for i, hook := range hooks.Hooks {
			wg.Add(1)
			go func(it int) {
				defer wg.Done()
				_ = hook(gracePeriod)
				log.Printf("Shutdown hook %d completed", it)
			}(i)
		}

		holdForWaitGroup := make(chan struct{})
		go func() {
			wg.Wait()
			logger.F("All shutdown hooks completed")
			close(holdForWaitGroup)
		}()

		select {
		case <-holdForWaitGroup:
			// Wait completed normally
		case <-time.After(gracePeriod):
			log.Printf("Shutdown hooks timed out after %v", gracePeriod)
		}
		logger.Info("Shutdown service done")
	}()

	return
}
