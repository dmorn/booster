package booster

import (
	"fmt"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

// Conn adds an identifier and a convenient RemoteNode field to a bare network.Conn.
type Conn struct {
	*network.Conn

	ID         string // ID is usually the remoteNode identifier.
	RemoteNode *node.Node
}

func (c *Conn) Close() error {
	return c.Conn.Close()
}

func (c *Conn) Send(p *packet.Packet) error {
	return c.Conn.Send(p)
}

// Recv reads packets from the underlying connection, without returning the packet if
// it is an heartbeat one.
func (c *Conn) Recv() (*packet.Packet, error) {
	// TODO(daniel): check if the packet is an heartbeat packet and handle
	// it accordingly
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
