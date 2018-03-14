package booster

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/pubsub"
	"github.com/danielmorandini/booster/socks5"
)

type Networks map[string]*Network

var Nets = &Networks{}

func (n Networks) Get(id string) *Network {
	net, ok := n[id]
	if !ok {
		panic("networks: tried to get unregistered network: " + id)
	}

	return net
}

func (n Networks) Set(id string, net *Network) {
	_, ok := n[id]
	if ok {
		panic("networks: tried to set already registered network: " + id)
	}

	net.boosterID = id
	n[id] = net
}

// Network describes a booster network: a local node, connected to other booster nodes
// using network.Conn as connector.
type Network struct {
	*log.Logger
	PubSub

	boosterID string
	IOTimeout time.Duration

	mux       sync.Mutex
	LocalNode *node.Node
	Conns     map[string]*Conn
}

func NewNet(n *node.Node, boosterID string) *Network {
	return &Network{
		Logger:    log.New(os.Stdout, "NETWORK  ", log.LstdFlags),
		PubSub:    pubsub.New(),
		LocalNode: n,
		boosterID: boosterID,
		IOTimeout: 2 * time.Second,
		Conns:     make(map[string]*Conn),
	}
}

// AddConn adds c to network. Returns an error if the connection is already present.
func (n *Network) AddConn(c *Conn) error {
	n.mux.Lock()
	defer n.mux.Unlock()

	if cc, ok := n.Conns[c.ID]; ok {
		// check if the connnection is nil. It that case, simply substitute
		// it.
		if cc.Conn != nil {
			return fmt.Errorf("network: conn (%v) already present", c.ID)
		}
	}

	c.boosterID = n.boosterID
	n.Conns[c.ID] = c
	return nil
}

func (n *Network) Notify() (chan interface{}, error) {
	return n.Sub(TopicNodes)
}

func (n *Network) StopNotifying(c chan interface{}) {
	n.Unsub(c, TopicNodes)
}

func (n *Network) Nodes() (*node.Node, []*node.Node) {
	n.mux.Lock()
	defer n.mux.Unlock()

	root := n.LocalNode
	nodes := []*node.Node{}

	for _, c := range n.Conns {
		if c.RemoteNode.IsActive() && c.Conn != nil {
			nodes = append(nodes, c.RemoteNode)
		}
	}

	return root, nodes
}

// NodeOf returns the pointer of the node inside the network that has the same ID as
// the requested node.
func (n *Network) NodeOf(node *node.Node) (*node.Node, error) {
	if node.IsLocal() {
		if node.ID() != n.LocalNode.ID() {
			return nil, fmt.Errorf("network: unexpected node %v, wanted %v", node.ID(), n.LocalNode.ID())
		}

		return n.LocalNode, nil
	}

	conn, ok := n.Conns[node.ID()]
	if !ok {
		return nil, fmt.Errorf("network: couldn't find node %v", node.ID())
	}

	return conn.RemoteNode, nil
}

func (n *Network) Ack(node *node.Node, id string) error {
	n.Printf("network: acknoledging (%v) on node (%v)", id, node.ID())

	node, err := n.NodeOf(node)
	if err != nil {
		return err
	}

	if err := node.Ack(id); err != nil {
		return err
	}

	n.Pub(node, TopicNodes)
	return nil
}

func (n *Network) RemoveTunnel(node *node.Node, id string, acknoledged bool) error {
	n.Printf("booster: removing (%v) on node (%v)", id, node.ID())

	node, err := n.NodeOf(node)
	if err != nil {
		return err
	}

	if err := node.RemoveTunnel(id, acknoledged); err != nil {
		return err
	}

	n.Pub(node, TopicNodes)
	return nil
}

func (n *Network) AddTunnel(node *node.Node, target string) {
	node, err := n.NodeOf(node)
	if err != nil {
		return
	}

	if !node.IsLocal() {
		// add the tunnel also to the local node. Every tunnel passes
		// also trough it
		n.AddTunnel(n.LocalNode, target)
	}

	n.Printf("booster: adding tunnel (%v) to node (%v)", target, node.ID())

	node.AddTunnel(target)
	n.Pub(node, TopicNodes)
}

func (b *Booster) UpdateNode(node *node.Node, tm *socks5.TunnelMessage, acknoledged bool) error {
	if tm.Event == socks5.EventPush {
		if err := Nets.Get(b.ID).Ack(node, tm.Target); err != nil {
			return err
		}
	}

	if tm.Event == socks5.EventPop {
		if err := Nets.Get(b.ID).RemoveTunnel(node, tm.Target, acknoledged); err != nil {
			return err
		}
	}

	return nil
}

func (n *Network) NewConn(conn *network.Conn, node *node.Node, id string) *Conn {
	return &Conn{
		Conn:       conn,
		Logger:     n.Logger,
		RemoteNode: node,
		ID:         id,
		boosterID:  n.boosterID,
		IOTimeout:  n.IOTimeout,
	}
}

// Conn adds an identifier and a convenient RemoteNode field to a bare network.Conn.
type Conn struct {
	*network.Conn
	*log.Logger

	ID             string // ID is usually the remoteNode identifier.
	boosterID      string
	RemoteNode     *node.Node
	IOTimeout      time.Duration
	HeartbeatTimer *time.Timer
}

// Close closes the connection and sets the status of the remote node
// to inactive and removes the connection from the network.
func (c *Conn) Close() error {
	c.Printf("network: closing conn (%v)", c.ID)

	if c.Conn == nil {
		return fmt.Errorf("network: connection is closed")
	}

	if err := c.Conn.Close(); err != nil {
		return err
	}
	c.RemoteNode.SetIsActive(false)

	n := Nets.Get(c.boosterID)
	// Remove the connection only if it is actually part of this network
	if _, ok := n.Conns[c.ID]; ok {
		n.Conns[c.ID].Conn = nil
	}

	n.Pub(c.RemoteNode, TopicNodes)

	return nil
}

func (c *Conn) Send(p *packet.Packet) error {
	if c.Conn == nil {
		return fmt.Errorf("network: connection is closed")
	}

	errc := make(chan error, 1)

	go func() {
		errc <- c.Conn.Send(p)
	}()

	select {
	case err := <-errc:
		return err
	case <-time.After(time.Second * 2):
		c.Close()
		return fmt.Errorf("network: couldn't send after %v", c.IOTimeout)
	}
}

func (c *Conn) Consume() (<-chan *packet.Packet, error) {
	if c.Conn == nil {
		return nil, fmt.Errorf("network: connection is closed")
	}
	return c.Conn.Consume()
}

func (c *Conn) Recv() (*packet.Packet, error) {
	if c.Conn == nil {
		return nil, fmt.Errorf("network: connection is closed")
	}
	return c.Conn.Recv()
}

func ValidatePacket(p *packet.Packet) error {
	// Find header
	hraw, err := p.Module(protocol.ModuleHeader)
	if err != nil {
		return err
	}

	h, err := protocol.DecodeHeader(hraw.Payload())
	if err != nil {
		return err
	}

	// Check packet version
	if !protocol.IsVersionSupported(h.ProtocolVersion) {
		return fmt.Errorf("packet validation: version (%v) is not supported", h.ProtocolVersion)
	}

	// Check that the information contained in the header reflect the
	// actual content of the packet
	for _, mid := range h.Modules {
		if _, err := p.Module(mid); err != nil {
			return fmt.Errorf("packet validation: %v", err)
		}
	}

	return nil
}

func ExtractHeader(p *packet.Packet) (*protocol.Header, error) {
	if err := ValidatePacket(p); err != nil {
		return nil, fmt.Errorf("booster: discarding invalid packet: %v", err)
	}

	// extract header from packet
	hraw, err := p.Module(protocol.ModuleHeader)
	if err != nil {
		return nil, fmt.Errorf("booster: failed reading module header: %v", err)
	}
	h, err := protocol.DecodeHeader(hraw.Payload())
	if err != nil {
		return nil, fmt.Errorf("booster: failed decoding header: %v", err)
	}

	return h, nil
}

func (b *Booster) composeHeartbeat(pl *protocol.PayloadHeartbeat) (*packet.Packet, error) {
	if pl == nil {
		pl = &protocol.PayloadHeartbeat{
			Hops: 0,
			ID:   "heartbeat", // TODO(daniel): unused field
		}
	}

	pl.Hops++
	pl.TTL = time.Now().Add(b.HeartbeatTTL / 2)

	h, err := protocol.HeartbeatHeader()
	if err != nil {
		return nil, err
	}
	hpl, err := protocol.EncodePayloadHeartbeat(pl)
	if err != nil {
		return nil, err
	}

	// compose the packet
	p := packet.New()
	enc := protocol.EncodingProtobuf
	if _, err := p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return nil, err
	}
	if _, err := p.AddModule(protocol.ModulePayload, hpl, enc); err != nil {
		return nil, err
	}

	return p, nil
}

func composeNode(n *node.Node) (*packet.Packet, error) {
	h, err := protocol.NodeHeader()
	if err != nil {
		return nil, err
	}

	tunnels := []*protocol.Tunnel{}
	for _, t := range n.Tunnels() {
		if t == nil {
			continue
		}

		tunnel := &protocol.Tunnel{
			ID:     t.ID(),
			Target: t.Target,
			Acks:   t.Acks(),
			Copies: t.Copies(),
		}

		tunnels = append(tunnels, tunnel)
	}
	param := &protocol.PayloadNode{
		ID:      n.ID(),
		BAddr:   n.BAddr.String(),
		PAddr:   n.PAddr.String(),
		Active:  n.IsActive(),
		Tunnels: tunnels,
	}

	npl, err := protocol.EncodePayloadNode(param)
	if err != nil {
		return nil, err
	}

	p := packet.New()
	enc := protocol.EncodingProtobuf
	if _, err = p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return nil, err
	}
	if _, err = p.AddModule(protocol.ModulePayload, npl, enc); err != nil {
		return nil, err
	}

	return p, nil
}
