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
	id      []byte // sha1
	BAddr   net.Addr
	PAddr   net.Addr
	isLocal bool

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
		return nil, errors.New("node: unable to create baddr: " + err.Error())
	}
	n.BAddr = baddr

	paddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, pport))
	if err != nil {
		return nil, errors.New("node: unable to create paddr: " + err.Error())
	}
	n.PAddr = paddr

	n.tunnels = make(map[string]*Tunnel)
	n.stop = make(chan struct{})
	n.id = sha1Hash([]byte(host), []byte(bport), []byte(pport))
	n.isLocal = isLocal

	return n, nil
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
	return fmt.Sprintf("%x", n.id)
}

// ProxyAddr returns the proxy address of the node.
func (n *Node) ProxyAddr() net.Addr {
	return n.PAddr
}

// AddTunnel sets the node's state to active and adds a new
// tunnel to it. If the node as already a tunnel with this
// target connected to it, it increments the copies of the
// tunnel.
func (n *Node) AddTunnel(target net.Addr) string {
	n.SetIsActive(true)
	nt := NewTunnel(target)

	if t, ok := n.tunnels[nt.ID()]; ok {
		t.Lock()
		defer t.Unlock()

		t.copies++
		return t.ID()
	}

	n.Lock()
	defer n.Unlock()
	n.tunnels[nt.ID()] = nt

	return nt.ID()
}

// Ack acknoledges the target tunnel, impling that the node is actually working on it.
func (n *Node) Ack(id string) error {
	n.Lock()
	t, ok := n.tunnels[id]
	if !ok {
		return fmt.Errorf("node: cannot ack [%v], no such tunnel", id)
	}
	n.Unlock()

	t.Lock()
	defer t.Unlock()

	if t.acks >= t.copies {
		return fmt.Errorf("node: cannot ack already acknoledged node [%v]: acks %v, copies: %v", id, t.acks, t.copies)
	}

	t.acks++
	return nil
}

func (n *Node) RemoveTunnel(id string) error {
	n.Lock()
	defer n.Unlock()

	t, ok := n.tunnels[id]
	if !ok {
		return fmt.Errorf("node: cannot delete [%v], no such tunnel", id)
	}

	t.Lock()
	defer t.Unlock()
	if t.copies == 1 {
		delete(n.tunnels, id)
		return nil
	}

	t.copies--
	return nil
}

func (n *Node) Tunnel(id string) (*Tunnel, error) {
	n.Lock()
	defer n.Unlock()

	t, ok := n.tunnels[id]
	if !ok {
		return nil, fmt.Errorf("node: no such tunnel [%v]", id)
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

func (n *Node) Stop() chan struct{} {
	n.Lock()
	defer n.Unlock()

	return n.stop
}

// Desc returns the description of the node in a multiline string.
func (n *Node) String() string {
	activeStr := "inactive"
	if n.IsActive() {
		activeStr = "active"
	}

	host, bport, _ := net.SplitHostPort(n.BAddr.String())
	_, pport, _ := net.SplitHostPort(n.PAddr.String())

	return fmt.Sprintf("[node (%v), @%v(b%v-p%v), %v]: wl: %v", n.ID(), host, bport, pport, activeStr, n.Workload())
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

func sha1Hash(images ...[]byte) []byte {
	h := sha1.New()
	for _, image := range images {
		h.Write(image)
	}

	return h.Sum(nil)
}
