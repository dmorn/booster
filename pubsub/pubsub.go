// Package pubsub provides the core functionalities to handle
// publication/subscription pipelines.
package pubsub

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PubSub wraps the core pubsub functionalities.
type PubSub struct {
	sync.Mutex
	registry map[string]*medium
}

// New returns a new PubSub instance.
func New() *PubSub {
	return &PubSub{
		registry: make(map[string]*medium),
	}
}

// Links returns the active channels related to topic.
func (ps *PubSub) Links(topic string) ([]chan interface{}, error) {
	m, err := ps.medium(topic)
	if err != nil {
		return nil, err
	}

	return m.links, nil
}

// Sub makes a subscription to topic. Returns the channel where
// the messages will be sent to, which should not be closed. If doing so,
// the channel will be removed from the list of subscribed channels.
func (ps *PubSub) Sub(topic string) chan interface{} {
	ps.Lock()
	defer ps.Unlock()

	hash := hash(topic)
	l := make(chan interface{})

	m, err := ps.medium(topic)
	if err != nil {
		m = &medium{
			done:  make(chan struct{}),
			id:    hash,
			topic: topic,
		}
	}

	if m.links == nil {
		m.links = []chan interface{}{}
	}

	m.links = append(m.links, l)
	ps.registry[hash] = m

	return l
}

// Unsub removes c from the list of subscribed channels of topic.
// Returns an error if no such topic is present, or if the channel
// is already no longer in the subscription list.
func (ps *PubSub) Unsub(c chan interface{}, topic string) error {
	m, err := ps.medium(topic)
	if err != nil {
		return err
	}

	links := m.links
	for i, l := range links {
		if l == c {
			ps.closeChanSafe(l)
			m.links = append(m.links[:i], m.links[i+1:]...) // remove it from the list
			return nil
		}
	}

	return errors.New("pubsub: unsub error: unable to find channel")
}

func (ps *PubSub) closeChanSafe(c chan interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// tried to close c, which was already closed.
		}
	}()

	close(c)
}

// Close removes a topic and closes its related channels.
func (ps *PubSub) Close(topic string) error {
	ps.Lock()
	defer ps.Unlock()

	m, err := ps.medium(topic)
	if err != nil {
		return err
	}

	for _, l := range m.links {
		close(l)
	}

	delete(ps.registry, hash(topic))

	return nil
}

// Pub broadcasts the message to the listeners of topic.
// Retuns an error if no such topic is present, unsubscribes
// a channel if it is closed when sending to it. (i.e. causes a panic)
func (ps *PubSub) Pub(message interface{}, topic string) error {
	ps.Lock()
	defer ps.Unlock()

	m, err := ps.medium(topic)
	if err != nil {
		return err
	}

	ps.broadcast(message, m)
	return nil
}

func (ps *PubSub) broadcast(message interface{}, medium *medium) {
	send := func(c chan interface{}) {
		defer func() {
			if r := recover(); r != nil {
				// remove the channel that caused the panic
				ps.Unsub(c, medium.topic)
			}
		}()

		select {
		case c <- message:
		case <-time.After(time.Second * 5):
		}
	}

	for _, l := range medium.links {
		go send(l)
	}
}

func (ps *PubSub) medium(topic string) (*medium, error) {
	hash := hash(topic)
	m, ok := ps.registry[hash]
	if !ok {
		return nil, errors.New("pubsub: topic " + topic + " not found")
	}

	return m, nil
}

type medium struct {
	id    string
	topic string
	done  chan struct{}
	links []chan interface{}
}

func (m *medium) String() string {
	return "medium (" + m.id + "): topic: " + m.topic
}

func hash(images ...string) string {
	h := sha1.New()
	for _, image := range images {
		h.Write([]byte(image))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
