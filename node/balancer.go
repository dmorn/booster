package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

// LoadBalancer is the interface that describes a load balancer object.
// It exposes methods to add and remove items, referenced by a string identifier.
type LoadBalancer interface {
	// GetBalanced should return a previously added item, using internally a
	// balancing algorithm.
	// trg should be used to set a minimum target requirement.
	GetBalanced(trg int) (addr string, err error)
	Add(addr string, conn Conn)
	Remove(addr string)
}

type entry struct {
	conn   Conn
	cancel context.CancelFunc

	sync.Mutex
	workload int
}

// Balancer is a LoadBalancer implementation
type Balancer struct {
	entries map[string]*entry
}

// NewBalancer returns a new balancer instance.
func NewBalancer() *Balancer {
	b := new(Balancer)
	b.entries = make(map[string]*entry)

	return b
}

// GetBalanced collects the workload its registered entries,
// and compares them to the trg workload, that represents the
// workload that the remote entry is supposed to beat in order
// to be used next.
//
// Returns an error if no candidate is found, either because
// none was provided or because no entry's workload was under
// the target.
func (b *Balancer) GetBalanced(trg int) (string, error) {
	var candidate *entry
	var addr string
	var twl int // total workload

	for key, e := range b.entries {
		if candidate == nil {
			candidate = e
			addr = key
		}

		e.Lock()
		ewl := e.workload // entry workload
		twl += ewl
		e.Unlock()

		candidate.Lock()
		cwl := candidate.workload // candidate workload
		candidate.Unlock()

		if ewl < cwl {
			candidate = e
			addr = key
		}
	}

	if candidate == nil {
		return "", errors.New("booster balancer: no remote boosters connected")
	}

	// trg is the sum of the local workload and the remote node's workload.
	// this is why we have to subtract the total remote workload to understand
	// how is the load on this node.
	if candidate.workload > (trg - twl) {
		return "", errors.New("booster balancer: use local proxy")
	}

	return addr, nil
}

// Add adds a new entry to the monitored proxies. conn is expected to
// come from a booster node, addr the proxy address.
func (b *Balancer) Add(addr string, conn Conn) {
	fmt.Printf("[BALANCER]: adding proxy %v\n", addr)

	ctx, cancel := context.WithCancel(context.Background())
	e := new(entry)
	e.conn = conn
	e.cancel = cancel

	if _, ok := b.entries[addr]; ok {
		// remove it and substitute
		b.Remove(addr)
	}

	b.entries[addr] = e

	// keep on updating entry's workload
	go func() {
		buf := make([]byte, 3)
		c := make(chan error)

		go func() {
			for {
				if _, err := io.ReadFull(conn, buf); err != nil {
					// unable to update status or something. Remove proxy?
					fmt.Printf("[BALANCER]: unable to update status: %v\n", err)
					b.Remove(addr)
					c <- err
					return
				}

				_ = buf[0]     // version - already checked in the hello procedure
				_ = buf[1]     // reserved field
				load := buf[2] // workload

				e.Lock()
				fmt.Printf("[BALANCER]: changing workload (%v) from %v to %v\n", e.conn.RemoteAddr(), e.workload, load)
				e.workload = int(load)
				e.Unlock()
			}
		}()

		// in any case we're done
		select {
		case <-ctx.Done():
			b.Remove(addr)
			return
		case <-c:
			return
		}
	}()
}

// Remove removes the entry labeled with addr. It expects addr
// to be the node's proxy address.
func (b *Balancer) Remove(addr string) {
	fmt.Printf("[BALANCER]: removing proxy %v\n", addr)

	// first stop updating its workload
	if e, ok := b.entries[addr]; ok {
		e.cancel()
		e.conn.Close()
	}

	delete(b.entries, addr)
}
