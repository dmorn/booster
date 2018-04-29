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

// Package tracer provides basic functionalities to monitor a network address
// until it is online.
package tracer

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/pubsub"
)

// Topic used to publish connectin discovery messgages.
const (
	TopicConn = "topic_connection"
)

// Possible Tracer status value.
const (
	StatusRunning = iota
	StatusStopped
)

// Possible connection states.
const (
	ConnOnline = iota
	ConnOffline
)

// Pinger wraps the basic Ping function.
type Pinger interface {
	Addr() net.Addr
	Ping(ctx context.Context) error
	ID() string
}

// PubSub describes the required functionalities of a publication/subscription object.
type PubSub interface {
	Sub(cmd *pubsub.Command) (pubsub.CancelFunc, error)
	Pub(message interface{}, topic string) error
}

// Tracer can monitor remote interfaces until they're up.
type Tracer struct {
	PubSub

	refreshc    chan struct{}
	stopc       chan struct{}
	conns       map[string]Pinger
	RefreshRate time.Duration

	sync.Mutex
	status int
}

type Message struct {
	ID  string
	Err error
}

// New returns a new instance of Tracer.
func New() *Tracer {
	t := &Tracer{
		PubSub:      pubsub.New(),
		conns:       make(map[string]Pinger),
		refreshc:    make(chan struct{}),
		stopc:       make(chan struct{}),
		status:      StatusStopped,
		RefreshRate: time.Second * 4,
	}

	return t
}

// Run makes the tracer listen for refresh calls and perform ping operations
// on each connection that is labeled with pending.
// Quits immediately when Close is called, runs in its own gorountine.
func (t *Tracer) Run() error {
	if t.Status() == StatusRunning {
		return errors.New("tracer: already running")
	}
	t.setStatus(StatusRunning)

	ping := func() context.CancelFunc {
		ctx, cancel := context.WithCancel(context.Background())

		for _, c := range t.conns {
			go func(c Pinger) {
				err := c.Ping(ctx)
				m := Message{ID: c.ID(), Err: err}

				if t.PubSub != nil {
					t.Pub(m, TopicConn)
				}
			}(c)
		}

		return cancel
	}

	go func() {
		var cancel context.CancelFunc
		for {
			refresh := func() {
				if cancel != nil {
					cancel()
				}
				cancel = ping()
			}

			select {
			case <-t.refreshc:
				refresh()
			case <-t.stopc:
				if cancel != nil {
					cancel()
				}
				return
			case <-time.After(t.RefreshRate):
				refresh()
			}
		}
	}()

	return nil
}

// Trace makes the tracer keep track of the entity at addr.
func (t *Tracer) Trace(p Pinger) error {
	log.Debug.Printf("tracer: tracing connection @ %v (%v)", p.Addr().String(), p.ID())
	t.conns[p.ID()] = p
	t.refresh()

	return nil
}

// Status returns the status of tracer.
func (t *Tracer) Status() int {
	t.Lock()
	defer t.Unlock()
	return t.status
}

func (t *Tracer) setStatus(status int) {
	t.Lock()
	defer t.Unlock()
	t.status = status
}

// Untrace removes the entity stored with id from the monitored
// entities.
func (t *Tracer) Untrace(id string) {
	log.Debug.Printf("tracer: untracing connection %v", id)

	delete(t.conns, id)
	t.refresh()
}

func (t *Tracer) refresh() {
	log.Debug.Printf("tracer: refreshing. connections monitored: %v", len(t.conns))
	t.refreshc <- struct{}{}
}

// Close makes the tracer pass from status running to status stopped.
func (t *Tracer) Close() {
	log.Debug.Printf("tracer: stopping.")
	t.setStatus(StatusStopped)
	t.stopc <- struct{}{}
}
