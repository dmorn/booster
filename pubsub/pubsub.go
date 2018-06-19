/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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

type CancelFunc func() error

type Action func(interface{}) error

// Command contains fields used in a subscription pipeline.
type Command struct {
	// Topic is the subscription topic.
	Topic string

	// Run will be called each time that the pubsub receives
	// new data on Topic.
	Run Action

	// PostRun is triggered before terminating the subscription
	// pipeline.
	PostRun func(error)

	// Ref is filled with the subscription index reference.
	Ref int
}

// Sub subscribes cmd to cmd.Topic in the pubsub. If there is no such
// topic it creates it. Returns a cancel function that can be used to
// terminate the subscription.
// cmd.Run gets called for each data coming from the pubsub. PostRun
// gets as argument the returned value of Run.
// Returns an error if the command does not contain at least a topic
// and a Run function.
func (ps *PubSub) Sub(cmd *Command) (CancelFunc, error) {
	// first check if the command contains at last a Run function
	// and a topic, otherwise it does not make any sense to
	// subscribe it.
	if cmd.Topic == "" {
		return nil, fmt.Errorf("pubsub: sub: no Topic provided")
	}
	if cmd.Run == nil {
		return nil, fmt.Errorf("pubsub: sub: no Run function")
	}

	tname := cmd.Topic
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

	// find free place
	t.Lock()
	ok := false
	index := 0
	for i, v := range t.chs {
		if v == nil {
			ok = true
			index = i
			t.chs[i] = ch
			break
		}
	}
	t.Unlock()

	if !ok {
		return nil, errors.New("pubsub: too many subscribers")
	}

	c, err := ch.run()
	if err != nil {
		return nil, err
	}

	go func() {
		var err error
		for d := range c {
			if err = cmd.Run(d); err != nil {
				// unsub closes c
				ps.Unsub(index, tname)
			}
		}
		if cmd.PostRun != nil {
			cmd.PostRun(err)
		}
	}()

	return func() error {
		return ps.Unsub(index, tname)
	}, nil
}

// Unsub removes c from the list of subscribed channels of topic.
// Returns an error if no such topic is present, or if the channel
// is already no longer in the subscription list.
func (ps *PubSub) Unsub(index int, topic string) error {
	t, err := ps.topic(topic)
	if err != nil {
		return err
	}

	t.Lock()
	defer t.Unlock()
	if index < 0 || index > len(t.chs) {
		return fmt.Errorf("pubsub: index out of range: %v, max: %v, topic: %v", index, len(t.chs), topic)
	}

	ch := t.chs[index]
	if ch == nil {
		return fmt.Errorf("pubsub: found nil channel at index: %v", index)
	}

	ch.stop()
	t.chs[index] = nil

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
// Returns an error if no such topic is present, unsubscribes
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
