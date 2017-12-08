package node

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// LoadBalancer is the interface that describes a load balancer object.
// It exposes methods to add and remove nodes.
type LoadBalancer interface {
	// GetNodeBalanced should returns a node id, using internally a
	// balancing algorithm.
	// tr should be used to set a minimum treshold requirement.
	GetNodeBalanced(tr int) (id string, err error)
	GetProxy(id string) (addr string, err error)
	AddNode(host, pport, bport string, conn net.Conn) (string, error)
	RemoveNode(id string) bool
}

type entry struct {
	id       string // sha1 string representation
	host     string
	pport    string
	bport    string
	isActive bool // set to false when connection is nil - for testing purposes only
	conn     net.Conn
	cancel   context.CancelFunc

	sync.Mutex
	workload int
}

// Balancer is a LoadBalancer implementation
type Balancer struct {
	*log.Logger

	entries map[string]*entry
}

// NewBalancer returns a new balancer instance.
func NewBalancer(log *log.Logger) *Balancer {
	b := new(Balancer)
	b.Logger = log
	b.entries = make(map[string]*entry)

	return b
}

// GetProxy returns the proxy address associated with id.
// Returns an error if no entry with this id is found.
func (b *Balancer) GetProxy(id string) (addr string, err error) {
	if e, ok := b.entries[id]; ok {
		return net.JoinHostPort(e.host, e.pport), nil
	}

	return "", errors.New("balancer: " + id + " not found")
}

// GetNodeBalanced collects the workload of its registered nodes,
// and compares them to the tr workload, that represents the
// workload that the remote node is supposed to "beat" in order
// to be used next.
//
// Returns an error if no candidate is found, either because
// none was provided or because no entry's workload was under
// the treshold.
func (b *Balancer) GetNodeBalanced(tr int) (string, error) {
	var c *entry // candidate entry
	var twl int  // total workload

	for _, e := range b.entries {
		if c == nil {
			c = e
		}

		e.Lock()
		ewl := e.workload
		twl += ewl
		e.Unlock()

		c.Lock()
		cwl := c.workload // candidate workload
		c.Unlock()

		if ewl < cwl && e.isActive {
			c = e
		}
	}

	if c == nil {
		return "", errors.New("booster balancer: no remote boosters connected")
	}

	// tr is the sum of the local workload and the remote node's workload.
	// this is why we have to subtract the total remote workload to understand
	// how is the load on this node.
	if c.workload > (tr - twl) {
		return "", errors.New("booster balancer: use local proxy")
	}

	return c.id, nil
}

// AddNode adds a new entry to the monitored nodes. conn is expected to
// come from a booster node.
// Returns the entry identifier.
func (b *Balancer) AddNode(host, pport, bport string, conn net.Conn) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	e := new(entry)
	e.host = host
	e.pport = pport
	e.bport = bport
	e.conn = conn
	e.cancel = cancel

	// id is the sha1 of host + bport + pport
	h := sha1.New()
	h.Write([]byte(e.host))
	h.Write([]byte(e.bport))
	h.Write([]byte(e.pport))
	e.id = fmt.Sprintf("%x", h.Sum(nil))

	if _, ok := b.entries[e.id]; ok {
		// remove it and substitute
		b.RemoveNode(e.id)
	}

	if conn == nil {
		e.isActive = false
	} else {
		e.isActive = true
	}

	b.Printf("balancer: adding proxy %v (%v) (active: %v)\n", e.id, net.JoinHostPort(e.host, e.pport), e.isActive)
	b.entries[e.id] = e

	// keep on updating entry's workload
	go func(e *entry) {
		if e.conn == nil {
			return
		}
		buf := make([]byte, 4)
		c := make(chan error)

		go func(e *entry) {
			for {
				if _, err := io.ReadFull(e.conn, buf); err != nil {
					// unable to update status or something. Remove proxy?
					b.Printf("balancer: unable to update status: %v\n", err)
					b.RemoveNode(e.id)
					c <- err
					return
				}

				_ = buf[0]     // version - already checked in the hello procedure
				_ = buf[1]     // cmd
				_ = buf[2]     // reserved field
				load := buf[3] // workload

				e.Lock()
				b.Printf("balancer: changing workload (%v) from %v to %v\n", e.conn.RemoteAddr(), e.workload, load)
				e.workload = int(load)
				e.Unlock()
			}
		}(e)

		// in any case we're done
		select {
		case <-ctx.Done():
			b.RemoveNode(e.id)
			return
		case <-c:
			return
		}
	}(e)

	return e.id, nil
}

// RemoveNode removes the entry labeled with id.
// Returns false if no entry was found.
func (b *Balancer) RemoveNode(id string) bool {
	if e, ok := b.entries[id]; ok {
		b.Printf("balancer: removing proxy %v\n", id)

		e.cancel()
		e.conn.Close()
		delete(b.entries, id)
		return ok
	}

	b.Printf("balancer: %v not found\n", id)
	return false
}
