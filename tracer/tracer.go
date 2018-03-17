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

	"github.com/danielmorandini/booster/pubsub"
)

// Topic used to publish connectin discovery messgages.
const (
	TopicConnDiscovered = "topic_conn_disc"
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
	Sub(topic string) (chan interface{}, error)
	Unsub(c chan interface{}, topic string) error
	Pub(message interface{}, topic string) error
}

// Tracer can monitor remote interfaces until they're up.
type Tracer struct {
	PubSub
	*log.Logger

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
		Logger:      log.New(os.Stdout, "TRACER   ", log.LstdFlags),
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
					t.Pub(m, TopicConnDiscovered)
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

func (t *Tracer) Notify() (chan interface{}, error) {
	return t.Sub(TopicConnDiscovered)
}

func (t *Tracer) StopNotifying(c chan interface{}) {
	t.Unsub(c, TopicConnDiscovered)
}

// Trace makes the tracer keep track of the entity at addr.
func (t *Tracer) Trace(p Pinger) error {
	t.Printf("tracer: tracing connection @ %v (%v)", p.Addr().String(), p.ID())
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
	t.setStatus(StatusStopped)
	t.stopc <- struct{}{}
}
