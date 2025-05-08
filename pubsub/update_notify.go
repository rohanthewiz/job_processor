package pubsub

import (
	"job_processor/shutdown"
	"sync"
	"time"

	"github.com/rohanthewiz/logger"
)

const (
	JobUpdateSubject = "job.update"
	CloseSignal      = "close"
)

// Singleton broker instance
var (
	defaultBroker *Broker
	once          sync.Once
)

// GetBroker returns the singleton broker
func GetBroker() *Broker {
	once.Do(func() {
		defaultBroker = NewBroker()
	})
	return defaultBroker
}

// StartPubSub initializes the pub-sub system
func StartPubSub() error {
	// Initialize the broker
	_ = GetBroker()
	return nil
}

// ListenForUpdates listens for updates and publishes them
func ListenForUpdates(updates <-chan any) error {
	broker := GetBroker()

	go func() {
		for update := range updates {
			broker.publish(JobUpdateSubject, update)
		}
	}()
	return nil
}

// SubscribeToUpdates subscribes to job updates
func SubscribeToUpdates(out chan any) (*Subscription, error) {
	broker := GetBroker()
	sub := broker.subscribe(JobUpdateSubject, out)

	shutdown.RegisterHook(func(_ time.Duration) error {
		logger.F("Shutting down subscription to %s on channel %v", JobUpdateSubject, out)
		sub.Unsubscribe()

		select {
		case out <- CloseSignal: // Notify subscriber to close
		default: // Don't block if out can't receive for some reason
		}
		return nil
	})

	return sub, nil
}
