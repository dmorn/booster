package booster

import (
	"fmt"
	"time"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

type SendConsumeCloser interface {
	SendCloser
	Consume() (<-chan *packet.Packet, error)
}

type SendCloser interface {
	Sender
	Closer
}

type Sender interface {
	Send(p *packet.Packet) error
}

type Closer interface {
	Close() error
}

// Conn adds an identifier and a convenient RemoteNode field to a bare network.Conn.
type Conn struct {
	*network.Conn

	ID         string // ID is usually the remoteNode identifier.
	RemoteNode *node.Node
}

// Close closes the connection and sets the status of the remote node
// to inactive.
func (c *Conn) Close() error {
	if err := c.Conn.Close(); err != nil {
		return err
	}
	c.RemoteNode.SetIsActive(false)

	return nil
}

func (c *Conn) Send(p *packet.Packet) error {
	return c.Conn.Send(p)
}

func (c *Conn) Consume() (<-chan *packet.Packet, error) {
	return c.Conn.Consume()
}

func (c *Conn) Recv() (*packet.Packet, error) {
	return  c.Conn.Recv()
}

// Network describes a booster network: a local node, connected to other booster nodes
// using network.Conn as connector.
type Network struct {
	LocalNode *node.Node
	Conns     map[string]*Conn
}

func (n *Network) AddConn(c *Conn) error {
	if _, ok := n.Conns[c.ID]; ok {
		return fmt.Errorf("network: conn (%v) already present", c.ID)
	}

	n.Conns[c.ID] = c
	return nil
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
			ID: "heartbeat", // TODO(daniel): unused field
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

func (b *Booster) Nodes() (*node.Node, []*node.Node) {
	b.mux.Lock()
	defer b.mux.Unlock()

	root := b.Network.LocalNode
	nodes := []*node.Node{}

	for _, c := range b.Network.Conns {
		nodes = append(nodes, c.RemoteNode)
	}

	return root, nodes
}

func (b *Booster) Ack(node *node.Node, id string) error {
	b.Printf("booster: acknoledging (%v) on node (%v)", id, node.ID())

	if err := node.Ack(id); err != nil {
		return err
	}

	b.Pub(node, TopicNodes)
	return nil
}

func (b *Booster) RemoveTunnel(node *node.Node, id string, acknoledged bool) error {
	b.Printf("booster: removing (%v) on node (%v)", id, node.ID())

	if err := node.RemoveTunnel(id, acknoledged); err != nil {
		return err
	}

	b.Pub(node, TopicNodes)
	return nil
}

func (b *Booster) AddTunnel(node *node.Node, target string) {
	b.Printf("booster: adding tunnel (%v) to node (%v)", target, node.ID())

	node.AddTunnel(target)
	b.Pub(node, TopicNodes)
}
