// Package pubsub provides the core functionalities to handle
// publication/subscription pipelines.
package pubsub

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"sync"
)

// PubSub wraps the core pubsub functionalities.
type PubSub struct {
	MaxSubs int // maximum number of subscribers

	sync.Mutex
	registry map[string]*topic
}

// New returns a new PubSub instance.
func New() *PubSub {
	return &PubSub{
		MaxSubs:  20,
		registry: make(map[string]*topic),
	}
}

// Sub makes a subscription to topic. Returns the channel where
// the messages will be sent to, which should not be closed. If doing so,
// the channel will be removed from the list of subscribed channels.
func (ps *PubSub) Sub(tname string) (chan interface{}, error) {
	hash := hash(tname)
	t, err := ps.topic(tname)
	if err != nil {
		t = &topic{
			id:   hash,
			name: tname,
			chs:  make([]*channel, ps.MaxSubs),
		}

		ps.Lock()
		ps.registry[hash] = t
		ps.Unlock()
	}

	ch := newChannel()

	t.Lock()
	defer t.Unlock()

	// find free place
	ok := false
	for i, v := range t.chs {
		if v == nil {
			ok = true
			t.chs[i] = ch
			break
		}
	}

	if !ok {
		return nil, errors.New("pubsub: too many subscribers")
	}

	return ch.out(), nil
}

// Unsub removes c from the list of subscribed channels of topic.
// Returns an error if no such topic is present, or if the channel
// is already no longer in the subscription list.
func (ps *PubSub) Unsub(c chan interface{}, topic string) error {
	t, err := ps.topic(topic)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()
	for i, ch := range t.chs {
		if ch == nil {
			continue
		}

		if ch.out() != c {
			continue
		}

		ch.stop()
		t.chs[i] = nil
	}

	return nil
}

// Close removes a topic and closes its related channels.
func (ps *PubSub) Close(topic string) error {
	t, err := ps.topic(topic)
	if err != nil {
		return err
	}

	t.Lock()
	for _, c := range t.chs {
		if c != nil {
			c.stop()
		}
	}
	t.Unlock()

	ps.Lock()
	delete(ps.registry, hash(topic))
	ps.Unlock()

	return nil
}

// Pub broadcasts the message to the listeners of topic.
// Retuns an error if no such topic is present, unsubscribes
// a channel if it is closed when sending to it. (i.e. causes a panic)
func (ps *PubSub) Pub(message interface{}, topic string) error {
	t, err := ps.topic(topic)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()

	for _, c := range t.chs {
		if c == nil {
			continue
		}

		c.send(message)
	}
	return nil
}

func (ps *PubSub) topic(name string) (*topic, error) {
	ps.Lock()
	defer ps.Unlock()

	hash := hash(name)
	m, ok := ps.registry[hash]
	if !ok {
		return nil, errors.New("pubsub: topic " + name + " not found")
	}

	return m, nil
}

type topic struct {
	id   string
	name string

	sync.Mutex
	chs []*channel
}

func (m *topic) String() string {
	return "topic (" + m.id + "): topic: " + m.name
}

func hash(images ...string) string {
	h := sha1.New()
	for _, image := range images {
		h.Write([]byte(image))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func closeChanSafe(c chan interface{}) {
	defer func() {
		if r := recover(); r != nil {
			// tried to close c, which was already closed.
		}
	}()

	close(c)
}
