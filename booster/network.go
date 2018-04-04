package booster

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/pubsub"
	"github.com/danielmorandini/booster/socks5"
	"github.com/danielmorandini/booster/tracer"
)

const TopicNodes = "topic_nodes"

// Networks is a convenience type that wraps a map of Network instances.
type Networks map[string]*Network

// Nets will be populated with the networks of every booster node started.
var Nets = &Networks{}

// Get returns a network associated with id. Panics if the network was not
// previously registered.
func (n Networks) Get(id string) *Network {
	net, ok := n[id]
	if !ok {
		panic("networks: tried to get unregistered network: " + id)
	}

	return net
}

// Set associates net with id inside of networks. Panics if the network
// labeled with id was alreaduy registered.
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
	PubSub
	*tracer.Tracer

	boosterID string
	IOTimeout time.Duration

	mux       sync.Mutex
	LocalNode *node.Node
	Conns     map[string]*Conn
}

func NewNet(n *node.Node, boosterID string) *Network {
	return &Network{
		PubSub:    pubsub.New(),
		Tracer:    tracer.New(),
		LocalNode: n,
		boosterID: boosterID,
		IOTimeout: 2 * time.Second,
		Conns:     make(map[string]*Conn),
	}
}

func (n *Network) TraceNodes(ctx context.Context) error {
	b, err := New(1111, 1111)
	if err != nil {
		return fmt.Errorf("network: tracer: %v", err)
	}

	if err := n.Tracer.Run(); err != nil {
		return fmt.Errorf("booster: trace nodes: %v", err)
	}

	errc := make(chan error)
	index, err := n.Tracer.Notify(func(i interface{}) {
		m, ok := i.(tracer.Message)
		if !ok {
			errc <- fmt.Errorf("booster: unable to recognise tracer message %v", m)
			return
		}

		// means that the device is still offline.
		if m.Err != nil {
			return
		}

		// find connection
		c, ok := n.Conns[m.ID]
		if !ok {
			log.Error.Printf("booster: tracer: found unresolved id: %v", m.ID)
			n.Untrace(m.ID)
			return
		}

		// reconnect to it
		from := n.LocalNode.BAddr.String()
		to := c.RemoteNode.BAddr.String()
		if _, err := b.Connect(ctx, "tcp", from, to); err != nil {
			// the node is up but we cannot open a proper Booster connection
			// to it.
			log.Error.Print(err)
			return
		}

		// do not trace this node anymore, as we managed to connect to it.
		n.Untrace(m.ID)
	})
	if err != nil {
		return fmt.Errorf("booster: trace nodes: %v", err)
	}

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		n.Tracer.StopNotifying(index)
		return ctx.Err()
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

		// Add the old tunnels to the new connection, as the node was not
		// manually disconnected.
		// Be aware that this operation could lead to the creation of
		// "zombie" tunnels, which are tunnels that are neither dead nor
		// alive.
		c.RemoteNode.CopyTunnels(cc.RemoteNode)
	}

	c.boosterID = n.boosterID
	n.Conns[c.ID] = c
	return nil
}

// Notify subscribes to the underlying pubsub with topic TopicNodes. The channel
// returned will produce node messages, containing information about the changes
// and updates performed in the network.
func (n *Network) Notify(f func(interface{})) (int, error) {
	return n.Sub(TopicNodes, f)
}

// StopNotifying usubscribes c from receiving notifications on TopicNodes.
// Closes the channel.
func (n *Network) StopNotifying(index int) {
	n.Unsub(index, TopicNodes)
}

// Nodes returns the root node of the network, togheter with all the remote
// nodes connected to it.
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

// Ack finds the node in the network and acknoledges the tunnel labeled
// with id. Publishes the node in TopicNodes.
func (n *Network) Ack(node *node.Node, id string) error {
	log.Debug.Printf("network: acknoledging (%v) on node (%v)", id, node.ID())

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

// RemoveTunnel finds the node in the network and removes the tunnel labeled
// with id from it. Publishes the node in TopicNodes.
func (n *Network) RemoveTunnel(node *node.Node, id string, acknoledged bool) error {
	log.Debug.Printf("booster: removing (%v) on node (%v)", id, node.ID())

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

// AddTunnel finds the node in the network, creates a new tunnel using target
// and adds the tunnel to the node. Publishes the node in TopicNodes.
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

	log.Debug.Printf("booster: adding tunnel (%v) to node (%v)", target, node.ID())

	node.AddTunnel(target)
	n.Pub(node, TopicNodes)
}

// UpdateNode acknoledges or remove a tunnel of node, depending on the tm's
// content.
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

// NewConn creates a new connection and associates it with the network.
func (n *Network) NewConn(conn *network.Conn, node *node.Node, id string) *Conn {
	return &Conn{
		Conn:       conn,
		RemoteNode: node,
		ID:         id,
		boosterID:  n.boosterID,
		IOTimeout:  n.IOTimeout,
	}
}

// Conn adds an identifier and a convenient RemoteNode field to a bare network.Conn.
type Conn struct {
	*network.Conn

	ID             string // ID is usually the remoteNode identifier.
	boosterID      string
	RemoteNode     *node.Node
	IOTimeout      time.Duration
	HeartbeatTimer *time.Timer
}

// Close closes the connection and sets the status of the remote node
// to inactive and removes the connection from the network.
func (c *Conn) Close() error {
	log.Debug.Printf("network: closing conn (%v)", c.ID)

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
	if c.RemoteNode.ToBeTraced {
		if err := n.Trace(c.RemoteNode); err != nil {
			return err
		}
	}

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

// TODO: what if consume accepted a function to be used for each packet? (see pubsub refactoring of
// Sub function)
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
	pl.TTL = time.Now().Add(b.HeartbeatTTL)

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
