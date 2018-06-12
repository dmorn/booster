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

// Package node provides functionalities for handling remote nodes and dispatching
// connections through them.
package node

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Node represents a remote booster node.
type Node struct {
	id         string
	BAddr      net.Addr
	PAddr      net.Addr
	isLocal    bool
	ToBeTraced bool

	sync.Mutex
	stop    chan struct{}
	active  bool // tells wether the node is updating its status or not
	tunnels map[string]*Tunnel
}

// New creates a new node instance.
func New(host, pport, bport string, isLocal bool) (*Node, error) {
	n := new(Node)
	baddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, bport))
	if err != nil {
		return nil, errors.New("New: unable to create baddr: " + err.Error())
	}
	n.BAddr = baddr

	paddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, pport))
	if err != nil {
		return nil, errors.New("New: unable to create paddr: " + err.Error())
	}
	n.PAddr = paddr

	n.tunnels = make(map[string]*Tunnel)
	n.stop = make(chan struct{})
	n.id = sha1Hash([]byte(host), []byte(bport), []byte(pport))
	n.isLocal = isLocal
	n.ToBeTraced = false

	return n, nil
}

// CopyTunnels takes the tunnels of the receiver and copies them into "into".
func (n *Node) CopyTunnels(into *Node) {
	n.Lock()
	into.Lock()
	defer n.Unlock()
	defer into.Unlock()

	for k, v := range n.tunnels {
		into.tunnels[k] = v
	}
}

// Workload is the number of tunnels that the node is managing. Contains also unacknoledged ones.
func (n *Node) Workload() int {
	n.Lock()
	defer n.Unlock()

	wl := 0
	for _, t := range n.tunnels {
		wl += t.Copies()
	}

	return wl
}

// IsLocal returns true if the node is a local node.
func (n *Node) IsLocal() bool {
	return n.isLocal
}

// SetIsActive sets the state of the node. Safe to be called by
// multiple goroutines.
func (n *Node) SetIsActive(active bool) {
	n.Lock()
	defer n.Unlock()

	n.active = active
}

// IsActive returns true if the node is updating it's status.
func (n *Node) IsActive() bool {
	n.Lock()
	defer n.Unlock()

	return n.active
}

// ID returns the node's sha1 identifier.
func (n *Node) ID() string {
	return n.id
}

// ProxyAddr returns the proxy address of the node.
func (n *Node) ProxyAddr() net.Addr {
	return n.PAddr
}

// PPort is a convenience method that returns the proxy port as string.
func (n *Node) PPort() string {
	_, p, _ := net.SplitHostPort(n.PAddr.String())
	return p
}

// BPort is a convenience method that returns the booster port as string.
func (n *Node) BPort() string {
	_, p, _ := net.SplitHostPort(n.BAddr.String())
	return p
}

// AddTunnel sets the node's state to active and adds a new
// tunnel to it. If the node as already a tunnel with this
// target connected to it, it increments the copies of the
// tunnel.
func (n *Node) AddTunnel(tunnel *Tunnel) {
	n.SetIsActive(true)

	n.Lock()
	defer n.Unlock()
	t, ok := n.tunnels[tunnel.Target]
	if ok {
		t.Lock()
		defer t.Unlock()

		t.copies++
		return
	}

	n.tunnels[tunnel.Target] = tunnel
}

func (n *Node) Tunnels() map[string]*Tunnel {
	n.Lock()
	defer n.Unlock()

	// make a copy of the map to avoid concurrent edit
	tcopy := make(map[string]*Tunnel)
	for k, v := range n.tunnels {
		tcopy[k] = v
	}

	return tcopy
}

func (n *Node) RemoveTunnel(target string, acknoledged bool) error {
	n.Lock()
	defer n.Unlock()

	t, ok := n.tunnels[target]
	if !ok {
		return fmt.Errorf("RemoveTunnel: tunnel %v does not exist in %v", target, n.ID())
	}

	t.Lock()
	defer t.Unlock()
	if t.copies == 1 {
		delete(n.tunnels, target)
		return nil
	}

	t.copies--

	return nil
}

func (n *Node) Tunnel(target string) (*Tunnel, error) {
	n.Lock()
	defer n.Unlock()

	t, ok := n.tunnels[target]
	if !ok {
		return nil, fmt.Errorf("Tunnel: %v not found", target)
	}

	return t, nil
}

func (n *Node) Close() error {
	n.SetIsActive(false)

	n.Lock()
	close(n.stop)
	n.Unlock()

	return nil
}

// Ping dials with the node with little timeout. Returns an error
// if the endpoint is not reachable, nil otherwise. Required by tracer.Pinger.
func (n *Node) Ping(ctx context.Context) error {
	if n.IsActive() {
		return errors.New("connection already enstablished")
	}

	d := net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 0 * time.Second,
	}
	_, err := d.DialContext(ctx, n.BAddr.Network(), n.BAddr.String())

	return err
}

func (n *Node) Addr() net.Addr {
	return n.BAddr
}

func sha1Hash(images ...[]byte) string {
	h := sha1.New()
	for _, image := range images {
		h.Write(image)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
