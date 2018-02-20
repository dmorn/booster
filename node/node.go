package node

import (
	"crypto/sha1"
	"encoding/hex"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
)

// Node represents a remote booster node.
type Node struct {
	id      []byte // sha1
	BAddr   net.Addr
	PAddr   net.Addr
	isLocal bool

	sync.Mutex
	stop chan struct{}
	active     bool // tells wether the node is updating its status or not
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

	return len(n.tunnels)
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
func (n *Node) AddTunnel(target net.Addr) {
	n.SetIsActive(true)

	if t, ok := n.tunnels[target.String()]; ok {
		t.Lock()
		defer t.Unlock()

		t.copies++
		return
	}

	t := NewTunnel(target)
	n.Lock()
	defer n.Unlock()
	n.tunnels[target.String()] = t
}

// Ack acknoledges the target tunnel, impling that the node is actually working on it.
func (n *Node) Ack(target net.Addr) error {
	n.Lock()
	t, ok := n.tunnels[target.String()]
	if !ok {
		return fmt.Errorf("node: cannot ack [%v], no such tunnel", target)
	}
	n.Unlock()

	t.Lock()
	defer t.Unlock()

	if t.acks >= t.copies {
		return fmt.Errorf("node: cannot ack already acknoledged node [%v]: acks %v, copies: %v", target, t.acks, t.copies)
	}

	t.acks++
	return nil
}

// Nack
func (n *Node) RemoveTunnel(target net.Addr) error {
	n.Lock()
	defer n.Unlock()

	t, ok := n.tunnels[target.String()]
	if !ok {
		return fmt.Errorf("node: cannot delete [%v], no such tunnel", target)
	}

	t.Lock()
	defer t.Unlock()
	if t.copies == 1 {
		delete(n.tunnels, target.String())
		return nil
	}

	t.copies--
	return nil
}

func (n *Node) Close() error {
	n.Lock()
	defer n.Unlock()

	n.SetIsActive(false)
	close(n.stop)
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

// Read  reads from reader expecting it to contain a node.
func (n *Node) Read(r io.Reader) error {
	buf := make([]byte, 20) // sha1 len
	if _, err := io.ReadFull(r, buf); err != nil {
		return errors.New("node: unable to read identifier: " + err.Error() + " buffer: " + fmt.Sprintf("%v", buf))
	}

	id := buf
	host, err := socks5.ReadHost(r)
	if err != nil {
		return errors.New("node: unable to decode host: " + err.Error())
	}
	pport, err := socks5.ReadPort(r)
	if err != nil {
		return errors.New("node: unable to decode p port: " + err.Error())
	}
	bport, err := socks5.ReadPort(r)
	if err != nil {
		return errors.New("node: unable to decode b port: " + err.Error())
	}

	buf = buf[:2]
	if _, err := io.ReadFull(r, buf); err != nil {
		return errors.New("node: unable to decode state: " + err.Error())
	}

	active := buf[0]
	count := int(buf[1])

	tns := make(map[string]*Tunnel)
	for i := 0; i < count; i++ {
		t := new(Tunnel)
		if err := t.Read(r); err != nil {
			return err
		}

		tns[t.ID()] = t
	}

	baddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, bport))
	if err != nil {
		return errors.New("node: unable to create baddr: " + err.Error())
	}
	paddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, pport))
	if err != nil {
		return errors.New("node: unable to create paddr: " + err.Error())
	}

	n.id = id
	n.BAddr = baddr
	n.PAddr = paddr
	n.active = active != 0
	n.tunnels = tns

	return nil
}

// EncodeBinary encodes the node into its binary
// representation.
func (n *Node) EncodeBinary() ([]byte, error) {
	if n == nil {
		return nil, errors.New("node: trying to encode nil")
	}

	host, bport, err := net.SplitHostPort(n.BAddr.String())
	_, pport, err := net.SplitHostPort(n.PAddr.String())
	if err != nil {
		return nil, errors.New("node: unable to split address: " + err.Error())
	}

	idbuf, err := hex.DecodeString(n.ID())
	hbuf, err := socks5.EncodeHostBinary(host)   // host buffer
	ppbuf, err := socks5.EncodePortBinary(pport) // proxy port buffer
	bpbuf, err := socks5.EncodePortBinary(bport) // booster port buffer
	if err != nil {
		return nil, errors.New("node: unable to encode: " + err.Error())
	}
	active := 0
	if n.IsActive() {
		active = 1
	}

	buf := make([]byte, 0, len(idbuf)+len(hbuf)+len(ppbuf)+len(bpbuf)+3+518)
	buf = append(buf, idbuf...)
	buf = append(buf, hbuf...)
	buf = append(buf, ppbuf...)
	buf = append(buf, bpbuf...)
	buf = append(buf, byte(active))
	buf = append(buf, byte(n.Workload()))

	n.Lock()
	defer n.Unlock()

	for _, t := range n.tunnels {
		tbuf, err := t.EncodeBinary()
		if err != nil {
			return nil, errors.New("node: unable to encode tunnel: " + err.Error())
		}

		buf = append(buf, tbuf...)
	}

	return buf, nil
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
