package pubsub

import (
	"sync"

	"github.com/rohanthewiz/logger"
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
	logger.F("Subscribed to topic: %s, channel: %v", topic, ch)

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

	chSubs, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for i, chSub := range chSubs {
		if chSub == ch {
			// Handle last element case properly
			if i == len(chSubs)-1 {
				b.subscribers[topic] = chSubs[:i]
			} else {
				b.subscribers[topic] = append(chSubs[:i], chSubs[i+1:]...)
			}
			break
		}
	}
	logger.F("Unsubscribed from topic: %s, channel: %v", topic, ch)
}

// Publish sends a message to all subscribers of a topic
func (b *Broker) publish(topic string, msg any) {
	b.mu.RLock()
	subs := b.subscribers[topic]
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- msg:

		// TODO: After n nbr of consecutive failed sends, unsubscribe
		default: // don't block if ch can't receive
		}
	}
}
