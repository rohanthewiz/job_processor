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

// InitShutdownServiceZ initializes the shutdown service.
// It will close the done channel to allow the app to shutdown
func InitShutdownServiceZ(done chan struct{}) {
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

		// Keep capturing signals so that subsequent CTRL-C's
		// 	don't kill us by default.
		go func() {
			// we're going to consume ALL future SIGINTs so they
			// don't fall through to the kernel's default.
			for sig := range sigChan {
				log.Printf("caught subsequent signal: %v", sig)

			}
		}()

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
		fmt.Println("Trying to sleep for 5 before exiting")
		time.Sleep(5 * time.Second) // Give time for signal to propagate
	}()

	return
}

func InitShutdownService(done chan struct{}) {
	sigChan := make(chan os.Signal, 1)

	// Notify for SIGINT and SIGTERM signals.
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Program is running. Press CTRL-C to exit.")

	// Wait for a signal in a goroutine.
	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal: %s. Waiting for graceful shutdown...\n", sig)
		// Simulate cleanup work.
		time.Sleep(10 * time.Second)
		fmt.Println("Cleanup complete.")
		close(done)
		// os.Exit(0)
	}()
}
