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

// Subscription represents a subscription to a topic
type Subscription struct {
	topic  string
	outCh  chan any
	broker *Broker
}

// Unsubscribe removes this subscription from the broker
func (s *Subscription) Unsubscribe() {
	s.broker.unsubscribe(s.topic, s.outCh)
}

// Broker manages topics and subscriptions
type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]chan any
}

// NewBroker creates a new message broker
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chan any),
	}
}

// Subscribe adds a new subscriber to a topic
func (b *Broker) subscribe(topic string, ch chan any) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], ch)
	logger.Info("Subscribed to topic: " + topic)

	return &Subscription{
		topic:  topic,
		outCh:  ch,
		broker: b,
	}
}

// Unsubscribe removes a subscriber from a topic
func (b *Broker) unsubscribe(topic string, ch chan any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for i, sub := range subs {
		if sub == ch {
			// Handle last element case properly
			if i == len(subs)-1 {
				b.subscribers[topic] = subs[:i]
			} else {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			}
			break
		}
	}
	logger.Info("Unsubscribed from topic: " + topic)
}

// Publish sends a message to all subscribers of a topic
func (b *Broker) publish(topic string, msg any) {
	b.mu.RLock()
	subs := b.subscribers[topic]
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- msg:
			// Message sent
		default:
			// Non-blocking send
		}
	}
}

// Global broker instance
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
		logger.Info("Shutting down subscription")
		sub.Unsubscribe()

		select {
		case out <- CloseSignal:
			// Close signal sent
		default:
			// Channel full or closed
		}

		return nil
	})

	return sub, nil
}
