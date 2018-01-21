// Package tracer provides basic functionalities to monitor a network address
// until it is online.
package tracer

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/danielmorandini/booster-network/pubsub"
)

// Topic used to publish connectin discovery messgages.
const (
	TopicConnDiscovered = "topic_conn_disc"
)

// Possible Tracer status value.
const (
	TrackerStatusRunning = iota
	TrackerStatusStopped
)

// Pinger wraps the basic Ping function.
type Pinger interface {
	net.Addr
	Ping(ctx context.Context) error
	ID() string
}

// Tracer can monitor remote interfaces until they're up.
type Tracer struct {
	*pubsub.PubSub
	*log.Logger

	refreshc chan struct{}
	stopc    chan struct{}
	conns    map[string]Pinger

	sync.Mutex
	status int
}

// New returns a new instance of Tracer. Calls Run before returning.
func New(lg *log.Logger, ps *pubsub.PubSub) *Tracer {
	t := &Tracer{
		Logger:   lg,
		PubSub:   ps,
		conns:    make(map[string]Pinger),
		refreshc: make(chan struct{}),
		stopc:    make(chan struct{}),
		status:   TrackerStatusStopped,
	}
	t.Run()

	return t
}

// NewDefault instantiates a new Tracer with default pubsub and logger.
func NewDefault() *Tracer {
	log := log.New(os.Stdout, "", log.LstdFlags)
	return New(log, pubsub.New())
}

// Run makes the tracer listen for refresh calls and perform ping operations
// on each connection that is labeled with pending.
// Quits immediately when Close is called, runs in its own gorountine.
func (t *Tracer) Run() error {
	if t.Status() == TrackerStatusRunning {
		return errors.New("tracer: already running")
	}
	t.setStatus(TrackerStatusRunning)

	ping := func() context.CancelFunc {
		ctx, cancel := context.WithCancel(context.Background())

		for _, c := range t.conns {
			go func(c Pinger) {
				if err := c.Ping(ctx); err == nil {
					// this connection resolves to an active connection
					t.Printf("tracer: found active connection @ %v (%v)", c.String(), c.ID())

					if t.PubSub != nil {
						t.Pub(c.ID(), TopicConnDiscovered)
					}
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
			case <-time.After(15 * time.Second):
				refresh()
			}
		}
	}()

	return nil
}

// Trace makes the tracer keep track of the entity at addr.
func (t *Tracer) Trace(p Pinger) error {
	t.Printf("tracer: tracing connection @ %v (%v)", p.String(), p.ID())
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
	t.Printf("tracer: untracing connection %v", id)

	delete(t.conns, id)
	t.refresh()
}

func (t *Tracer) refresh() {
	t.Printf("tracer: refreshing. connections monitored: %v", len(t.conns))
	t.refreshc <- struct{}{}
}

// Close makes the tracer pass from status running to status stopped.
func (t *Tracer) Close() {
	t.Printf("tracer: stopping.")
	t.setStatus(TrackerStatusStopped)
	t.stopc <- struct{}{}
}
