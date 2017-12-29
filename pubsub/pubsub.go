package pubsub

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"time"
)

type PubSub struct {
	registry map[string]*medium
}

func New() *PubSub {
	return &PubSub{
		registry: make(map[string]*medium),
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

func (ps *PubSub) Links(topic string) ([]link, error) {
	m, err := ps.medium(topic)
	if err != nil {
		return nil, err
	}

	return m.links, nil
}

func (ps *PubSub) Sub(topic string) chan interface{} {
	hash := hash(topic)
	m, err := ps.medium(topic)
	if err != nil {
		m = &medium{
			done: make(chan struct{}),
			id:   hash,
		}
		ps.registry[hash] = m
	}

	if m.links == nil {
		m.links = []link{}
	}

	l := make(chan interface{})
	m.links = append(m.links, l)

	return l
}

func (ps *PubSub) Unsub(c chan interface{}, topic string) error {
	m, err := ps.medium(topic)
	if err != nil {
		return err
	}

	links := m.links
	for i, l := range links {
		if l == c {
			close(m.links[i])
			m.links = append(m.links[:i], m.links[i+1:]...)
			return nil
		}
	}

	return errors.New("pubsub: unsub error: unable to find channel")
}

func (ps *PubSub) Close(topic string) error {
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

func (ps *PubSub) Pub(message interface{}, topic string) error {
	m, err := ps.medium(topic)
	if err != nil {
		return err
	}

	m.broadcast(message)
	return nil
}

type link chan interface{}

type medium struct {
	id    string
	done  chan struct{}
	links []link
}

func (m *medium) broadcast(data interface{}) {
	send := func(c chan interface{}) {
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()

		select {
		case c <- data:
		case <-time.After(time.Second * 5):
		}
	}

	for _, l := range m.links {
		go send(l)
	}
}

func (m *medium) String() string {
	return m.id
}

func hash(images ...string) string {
	h := sha1.New()
	for _, image := range images {
		h.Write([]byte(image))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
