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

package booster

import (
	"context"
	"encoding/json"
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

// Close gets the network associated with id, then it closes each connection
// stored in it. No remote node will be traced.
func (n Networks) Close(id string) {
	net := n.Get(id)

	for _, c := range net.Outgoing {
		c.RemoteNode.ToBeTraced = false
		c.Close()
	}
}

// Network describes a booster network: a local node, connected to other booster nodes
// using network.Conn as connector.
type Network struct {
	PubSub
	*tracer.Tracer

	boosterID    string
	IOTimeout    time.Duration
	HeartbeatTTL time.Duration
	DialTimeout  time.Duration

	mux       sync.Mutex
	LocalNode *node.Node
	Outgoing  map[string]*Conn
}

// NewNet returns a new network instance, configured with default parameters.
func NewNet(n *node.Node, boosterID string) *Network {
	return &Network{
		PubSub:    pubsub.New(),
		Tracer:    tracer.New(),
		LocalNode: n,
		boosterID: boosterID,
		IOTimeout: 2 * time.Second,
		Outgoing:  make(map[string]*Conn),
	}
}

// EncodeDefault calls Encode using protobuf for encoding, no compression nor encryption.
func (n *Network) EncodeDefault(v interface{}, msg protocol.Message) (*packet.Packet, error) {
	return n.Encode(v, msg, packet.Metadata{
		Encoding:    protocol.EncodingProtobuf,
		Compression: protocol.CompressionNone,
		Encryption:  protocol.EncryptionNone,
	})
}

func (n *Network) Encode(v interface{}, msg protocol.Message, meta packet.Metadata) (*packet.Packet, error) {
	// TODO: this could be the place where we encrypt the payloads
	p := packet.New()
	p.M = meta

	var f protocol.EncoderFunc
	if meta.Encoding == protocol.EncodingJson {
		f = json.Marshal
	} else {
		f = protocol.HeaderEncoder
	}

	// encode the header...
	hraw, err := protocol.Encode(msg, f)
	if err != nil {
		return nil, fmt.Errorf("network: %v", err)
	}
	// ... and add it to the packet
	if _, err = p.AddModule(string(protocol.ModuleHeader), hraw); err != nil {
		return nil, fmt.Errorf("network: encode failure: %v", err)
	}

	// encode the payload if present
	if v != nil {
		if meta.Encoding == protocol.EncodingJson {
			f = json.Marshal
		} else {
			f = protocol.PayloadEncoders[msg]
		}

		praw, err := protocol.Encode(v, f)
		if err != nil {
			return nil, fmt.Errorf("network: %v", err)
		}

		if _, err = p.AddModule(string(protocol.ModulePayload), praw); err != nil {
			return nil, fmt.Errorf("network: encode failure: %v", err)
		}
	}

	return p, nil
}

// Decode tries to decode the payload content of the module m contained
// in p into v, which has to be a pointer to a struct.
// f is the function that will be used for the decoding. It is important
// to choose the right combo between f and v, meaning that we cannot
// decode an "Hello" message using a "Node" decoder.
//
// Check package protocol for the decoding functions available.
func (n *Network) Decode(p *packet.Packet, m protocol.Module, v interface{}) error {
	// TODO: this is the place where we can decode a packet that was
	// encripted. Network should store the cookie (key), use it here
	// to descrypt the content of the packet and then decode the
	// payload.

	// validate packet
	if err := n.ValidatePacket(p); err != nil {
		return err
	}

	// find encoding type
	var f protocol.DecoderFunc
	if p.M.Encoding == protocol.EncodingJson {
		f = json.Unmarshal
	} else {
		f = protocol.HeaderDecoder
	}

	header := new(protocol.Header)
	if err := n.decode(p, protocol.ModuleHeader, &header, f); err != nil {
		return err
	}

	if p.M.Encoding == protocol.EncodingProtobuf {
		f = protocol.PayloadDecoders[header.ID]
	}

	return n.decode(p, m, v, f)
}

func (n *Network) decode(p *packet.Packet, m protocol.Module, v interface{}, f protocol.DecoderFunc) error {
	// extract the module
	mod, err := p.Module(string(m))
	if err != nil {
		return fmt.Errorf("network: decode error: %v", err)
	}

	// decode its payload into v
	err = protocol.Decode(mod.Payload(), v, f)
	if err != nil {
		return fmt.Errorf("network: %v", err)
	}

	return nil
}

func (n *Network) composeHeartbeat(pl *protocol.PayloadHeartbeat) (*packet.Packet, error) {
	var payload protocol.PayloadHeartbeat
	if pl == nil {
		payload = protocol.PayloadHeartbeat{
			Hops: 0,
			ID:   "heartbeat", // TODO(daniel): unused field
		}
	} else {
		payload = *pl
	}

	payload.Hops++
	payload.TTL = time.Now().Add(n.HeartbeatTTL)

	return n.EncodeDefault(payload, protocol.MessageHeartbeat)
}

func (n *Network) composeNode(node *node.Node) (*packet.Packet, error) {
	tunnels := []*protocol.Tunnel{}
	for _, t := range node.Tunnels() {
		if t == nil {
			continue
		}

		tunnel := &protocol.Tunnel{
			ID:        t.ID(),
			Target:    t.Target,
			ProxiedBy: t.ProxiedBy,
			Acks:      t.Acks(),
			Copies:    t.Copies(),
		}

		tunnels = append(tunnels, tunnel)
	}
	param := protocol.PayloadNode{
		ID:      node.ID(),
		BAddr:   node.BAddr.String(),
		PAddr:   node.PAddr.String(),
		Active:  node.IsActive(),
		Tunnels: tunnels,
	}

	return n.EncodeDefault(param, protocol.MessageNodeStatus)
}

// ValidatePackets extracts the header from the packet and checks if
// the validity/reliability of its contents.
func (n *Network) ValidatePacket(p *packet.Packet) error {
	// Find header
	m := protocol.ModuleHeader
	h := new(protocol.Header)
	f := protocol.HeaderDecoder
	if err := n.decode(p, m, &h, f); err != nil {
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

// TraceNodes starts a tracer that is able to determine wether a remote
// interface is up or down in the network. When a node get's traced is
// added back to the connection as soon as it online again.
func (n *Network) TraceNodes(ctx context.Context, b *Booster) error {
	if err := n.Tracer.Run(); err != nil {
		return fmt.Errorf("network: trace nodes: %v", err)
	}
	defer n.Tracer.Close()

	errc := make(chan error)
	cancel, err := n.Tracer.Sub(&pubsub.Command{
		Topic: tracer.TopicConn,
		Run: func(i interface{}) error {
			m, ok := i.(tracer.Message)
			if !ok {
				return fmt.Errorf("network: unable to recognise tracer message %v", m)
			}

			// means that the device is still offline.
			if m.Err != nil {
				return nil
			}

			// find connection
			c, ok := n.Outgoing[m.ID]
			if !ok {
				log.Error.Printf("network: tracer: found unresolved id: %v", m.ID)
				n.Untrace(m.ID)
				return nil
			}

			// reconnect to it
			from := n.LocalNode.BAddr.String()
			to := c.RemoteNode.BAddr.String()
			if _, err := b.Connect(ctx, "tcp", from, to); err != nil {
				// the node is up but we cannot open a proper Booster connection
				// to it.
				log.Error.Print(err)
				return nil
			}

			// do not trace this node anymore, as we managed to connect to it.
			n.Untrace(m.ID)
			return nil
		},
		PostRun: func(err error) {
			if err != nil {
				errc <- err
			}
		},
	})
	if err != nil {
		return fmt.Errorf("network: trace nodes: %v", err)
	}

	select {
	case err := <-errc:
		log.Error.Println(err)
		return err
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}

// AddConn adds c to network. Returns an error if the connection is already present.
func (n *Network) AddConn(c *Conn) error {
	n.mux.Lock()
	defer n.mux.Unlock()

	if cc, ok := n.Outgoing[c.ID]; ok {
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
		cc.RemoteNode.CopyTunnels(c.RemoteNode)
	}

	c.boosterID = n.boosterID
	n.Outgoing[c.ID] = c
	return nil
}

// Nodes returns the root node of the network, togheter with all the remote
// nodes connected to it.
func (n *Network) Nodes() (*node.Node, []*node.Node) {
	n.mux.Lock()
	defer n.mux.Unlock()

	root := n.LocalNode
	nodes := []*node.Node{}

	for _, c := range n.Outgoing {
		if c.RemoteNode.IsActive() && c.Conn != nil {
			nodes = append(nodes, c.RemoteNode)
		}
	}

	return root, nodes
}

// Ack finds the node in the network and acknoledges the tunnel labeled
// with target. Publishes the node in TopicNode.
func (n *Network) Ack(node *node.Node, target string) error {
	log.Debug.Printf("network: acknoledging (%v) on node (%v)", target, node.ID())

	if err := node.Ack(target); err != nil {
		return err
	}

	return nil
}

// RemoveTunnel finds the node in the network and removes the tunnel labeled
// with target from it. Publishes the node in TopicNode.
func (n *Network) RemoveTunnel(node *node.Node, target string, acknoledged bool) error {
	log.Debug.Printf("network: removing (%v) on node (%v)", target, node.ID())

	if err := node.RemoveTunnel(target, acknoledged); err != nil {
		return err
	}

	return nil
}

// AddTunnel finds the node in the network, creates a new tunnel using target
// and adds the tunnel to the node. If node is not a local node, it also
// adds the tunnel to the local node, but settings its ProxiedBy value to the
// remote node's proxy address. Publishes the update in TopicNode.
func (n *Network) AddTunnel(node *node.Node, t *node.Tunnel) {
	if !node.IsLocal() {
		t.ProxiedBy = node.ProxyAddr().String()
		n.AddTunnel(n.LocalNode, t.Copy())
	}

	log.Debug.Printf("network: adding tunnel (%v) to node (%v)", t.Target, node.ID())

	node.AddTunnel(t)
}

// UpdateNode acknoledges or removes a tunnel of node, depending on p's content.
func (b *Booster) UpdateNode(node *node.Node, p protocol.PayloadProxyUpdate, acknoledged bool) error {
	if !node.IsLocal() {
		p.ProxiedBy = node.ProxyAddr().String()
	}

	n := b.Net()
	n.Pub(p, socks5.TopicTunnelEvents)

	switch p.Operation {
	case protocol.TunnelAck:
		if err := n.Ack(node, p.Target); err != nil {
			return err
		}
	case protocol.TunnelRemove:
		if err := n.RemoveTunnel(node, p.Target, acknoledged); err != nil {
			return err
		}
	default:
		return fmt.Errorf("update node: unrecognised operation: %+v", p)
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

// Close takes care of closing the network connection that wires together the
// root node with its connected remote node.
func (c *Conn) Close() error {
	log.Debug.Printf("network: closing conn (%v)", c.ID)

	if c.Conn == nil {
		return fmt.Errorf("network: connection is closed")
	}

	// Close and remove the connection, so we can also check if c.Conn == nil
	if err := c.Conn.Close(); err != nil {
		return err
	}
	c.Conn = nil
	// Set remote node to inactive
	c.RemoteNode.SetIsActive(false)

	n := Nets.Get(c.boosterID)
	// Publish the event and trace the node only if requested. For example, we
	// do not want to trace a node that was manually disconnected.
	if c.RemoteNode.ToBeTraced {
		if err := n.Trace(c.RemoteNode); err != nil {
			return err
		}
	}

	return nil
}

// Send tries to send p though the connection. If the operation is not performed
// bofore IOTimeout, returns an error.
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
	case <-time.After(c.IOTimeout):
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
