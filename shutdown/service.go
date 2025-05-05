package shutdown

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const gracePeriod = 20 * time.Second

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
	fmt.Printf("Registered shutdown hook: %d\n", len(hooks.Hooks))
}

// InitShutdownService initializes the shutdown service.
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

		// Give manager time to shutdown gracefully
		log.Printf("Shutting down %d hooks grace period is: %s", len(hooks.Hooks), gracePeriod)

		for i, hook := range hooks.Hooks {
			wg.Add(1)
			go func(it int) {
				defer wg.Done()
				_ = hook(gracePeriod)
				log.Printf("Shutdown hook %d completed", it)
			}(i)

		}
		wg.Wait()

	}()

	return
}
