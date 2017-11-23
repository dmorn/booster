package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

type LoadBalancer interface {
	GetProxy() (string, error)
	AddProxy(addr string, conn Conn)
	LocalWorkload(chan int)
	RemoveProxy(addr string)
}

type entry struct {
	conn   Conn
	cancel context.CancelFunc

	sync.Mutex
	workload int
}

type Balancer struct {
	entries map[string]*entry

	sync.Mutex
	hasLocalLoad bool
	workload     int
}

func NewBalancer() *Balancer {
	b := new(Balancer)
	b.entries = make(map[string]*entry)

	return b
}

func (b *Balancer) GetProxy() (string, error) {
	var candidate *entry
	var addr string

	for key, e := range b.entries {
		if candidate == nil {
			candidate = e
			addr = key
		}

		e.Lock()
		twl := e.workload // test workload
		e.Unlock()

		candidate.Lock()
		cwl := candidate.workload // candidate workload
		candidate.Unlock()

		if twl < cwl {
			candidate = e
			addr = key
		}
	}

	if candidate == nil {
		return "", errors.New("booster balancer: no remote boosters connected")
	}

	b.Lock()
	defer b.Unlock()
	if b.hasLocalLoad {
		// if candidate has more load than the local proxy,
		// return an error and make the proxy use the local dialer
		if b.workload < candidate.workload {
			return "", errors.New("booster balancer: use local dialer")
		}
	}

	return addr, nil
}

func (b *Balancer) AddProxy(addr string, conn Conn) {
	fmt.Printf("[BALANCER]: adding proxy %v\n", addr)

	ctx, cancel := context.WithCancel(context.Background())
	e := new(entry)
	e.conn = conn
	e.cancel = cancel

	if _, ok := b.entries[addr]; ok {
		// remove it and substitute
		b.RemoveProxy(addr)
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
					b.RemoveProxy(addr)
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
			b.RemoveProxy(addr)
			return
		case <-c:
			return
		}
	}()
}

func (b *Balancer) RemoveProxy(addr string) {
	fmt.Printf("[BALANCER]: removing proxy %v\n", addr)

	// first stop updating its workload
	if e, ok := b.entries[addr]; ok {
		e.cancel()
		e.conn.Close()
	}

	delete(b.entries, addr)
}

func (b *Balancer) LocalWorkload(c chan int) {
	b.Lock()
	b.hasLocalLoad = true
	b.Unlock()

	go func(c chan int) {
		for w := range c {
			b.Lock()
			fmt.Printf("[BALANCER]: local workload updated from %v to %v\n", b.workload, w)
			b.workload = w
			b.Unlock()
		}

		b.Lock()
		b.hasLocalLoad = false
		b.Unlock()
	}(c)
}
