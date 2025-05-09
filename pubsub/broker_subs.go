package pubsub

import (
	"fmt"
	"sync"

	"github.com/rohanthewiz/logger"
)

// Subscription represents a subscription to a topic
type Subscription struct {
	topic  string
	chWrap chanWrap
	broker *Broker
}

type chanWrap struct {
	ch          chan any
	consecFails int
}

// Unsubscribe removes this subscription from the broker
func (s *Subscription) Unsubscribe() {
	s.broker.unsubscribe(s.topic, s.chWrap)
}

// Broker manages topics and subscriptions
type Broker struct {
	mu          sync.RWMutex
	subscribers map[string][]chanWrap
}

// NewBroker creates a new message broker
func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]chanWrap),
	}
}

// Subscribe adds a new subscriber to a topic
func (b *Broker) subscribe(topic string, cw chanWrap) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], cw)
	logger.F("Subscribed to topic: %s, channel: %v", topic, cw.ch)

	return &Subscription{
		topic:  topic,
		chWrap: cw,
		broker: b,
	}
}

// Unsubscribe removes a subscriber from a topic
func (b *Broker) unsubscribe(topic string, cw chanWrap) {
	b.mu.Lock()
	defer b.mu.Unlock()

	chSubs, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for i, chSub := range chSubs {
		if chSub.ch == cw.ch {
			// Handle last element case properly
			if i == len(chSubs)-1 {
				b.subscribers[topic] = chSubs[:i]
			} else {
				b.subscribers[topic] = append(chSubs[:i], chSubs[i+1:]...)
			}
			break
		}
	}
	logger.F("Unsubscribed from topic: %s, channel: %v", topic, cw.ch)
}

// Publish sends a message to all subscribers of a topic
func (b *Broker) publish(topic string, msg any) {
	b.mu.RLock()
	subs := b.subscribers[topic]
	b.mu.RUnlock()

	lnSubs := len(subs)
	for i := 0; i < lnSubs; i++ {
		fmt.Printf("Publishing to topic: %s, channel: %v\n", topic, subs[i].ch)
		fmt.Println("Current fails:", subs[i].consecFails)
		if subs[i].consecFails > 3 { // Perhaps make this configurable
			logger.F("Too many consecutive failures for topic: %s, channel: %v", topic, subs[i].ch)
			b.unsubscribe(topic, subs[i])
			break // unsubscribe changes the subs slice, so break to avoid issues
		}

		select {
		case subs[i].ch <- msg:
			subs[i].consecFails = 0 // reset consecutive failures

		default: // don't block if ch can't receiv
			fmt.Println("Incrementing fails...") // e
			subs[i].consecFails++
		}
	}
}
