package tracer

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/danielmorandini/booster-network/pubsub"
)

const (
	TopicConnDiscovered = "topic_conn_disc"
)

const (
	connStatusPending = iota
	connStatusDiscovered
)

const (
	TrackerStatusRunning = iota
	TrackerStatusStopped
)

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

func (t *Tracer) Trace(addr net.Addr, id string) error {
	t.Printf("tracer: tracing connection @ %v (%v)", addr.String(), id)

	c := &connection{
		id:     id,
		addr:   addr,
		status: TrackerStatusRunning,
	}
	t.conns[id] = c
	t.refresh()

	return nil
}

func (t *Tracer) Untrace(id string) {
	t.Printf("tracer: untracing connection %v", id)

	delete(t.conns, id)
	t.refresh()
}

func (t *Tracer) refresh() {
	t.Printf("tracer: refreshing. connections monitored: %v", len(t.conns))
	t.refreshc <- struct{}{}
}

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
