// Package tracer provides basic functionalities to monitor a network address
// until it is online.
package tracer

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/danielmorandini/booster-network/pubsub"
)

// Topic used to publish connectin discovery messgages.
const (
	TopicConnDiscovered = "topic_conn_disc"
)

const (
	connStatusPending = iota
	connStatusDiscovered
)

// Possible Tracer status value.
const (
	TrackerStatusRunning = iota
	TrackerStatusStopped
)

// Tracer can monitor remote interfaces until they're up.
type Tracer struct {
	*pubsub.PubSub
	*log.Logger

	status   int
	refreshc chan struct{}
	stopc    chan struct{}
	conns    map[string]*connection
}

// New returns a new instance of Tracer. Calls Run before returning.
func New(lg *log.Logger, ps *pubsub.PubSub) *Tracer {
	t := &Tracer{
		Logger: lg,
		PubSub: ps,
		conns:  make(map[string]*connection),
	}
	go t.Run()

	return t
}

// Run makes the tracer listen for refresh calls and perform ping operations
// on each connection that is labeled with pending.
// Quits immediately when Close is called.
func (t *Tracer) Run() error {
	t.status = TrackerStatusRunning

	done := make(chan struct{})
	ping := func() context.CancelFunc {
		ctx, cancel := context.WithCancel(context.Background())

		for _, c := range t.conns {
			go func(c *connection) {
				// do not consider non pending connections
				if c.status != connStatusPending {
					return
				}

				if err := c.ping(ctx); err == nil {
					// this connection resolves to an active connection
					c.status = connStatusDiscovered
					t.Printf("tracer: found active connection @ %v (%v)", c.addr.String(), c.id)

					if t.PubSub != nil {
						t.Pub(c.id, TopicConnDiscovered)
					}
				}
			}(c)
		}

		return cancel
	}

	go func() {
		for {
			var cancel context.CancelFunc
			refresh := func() {
				if cancel != nil {
					cancel()
				}
				cancel = ping()
			}

			select {
			case <-time.After(15 * time.Second):
				refresh()
			case <-t.refreshc:
				refresh()
			case <-t.stopc:
				if cancel != nil {
					cancel()
				}
				done <- struct{}{}
			}
		}
	}()

	<-done
	t.status = TrackerStatusStopped

	return nil
}

// Trace makes the tracer keep track of the entity at addr.
func (t *Tracer) Trace(addr net.Addr, id string) error {
	t.Printf("tracer: tracing connection @ %v (%v)", addr.String(), id)

	c := &connection{
		id:     id,
		addr:   addr,
		status: connStatusPending,
	}
	t.conns[id] = c
	t.refresh()

	return nil
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
	t.stopc <- struct{}{}
}

type connection struct {
	id     string
	addr   net.Addr
	status int
}

func (c *connection) ping(ctx context.Context) error {
	d := net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 0 * time.Second,
	}
	_, err := d.DialContext(ctx, c.addr.Network(), c.addr.String())

	return err
}
