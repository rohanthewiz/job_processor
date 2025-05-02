package shutdown

import (
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
}

func InitService(done chan struct{}) (sigChan chan os.Signal) {
	// Setup shutdown signal handling
	sigChan = make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Go handle shutdown signal
	go func() {
		sig := <-sigChan
		log.Printf("Received shutdown signal: %v", sig)
		setShutdown()

		// Give manager time to shutdown gracefully
		log.Printf("Shutting down... grace period is: %s", gracePeriod)

		for _, hook := range hooks.Hooks {
			_ = hook(gracePeriod)
		}

		close(done)
	}()

	return
}
