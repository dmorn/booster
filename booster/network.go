package booster

import (
	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/node"
)

// Conn adds an identifier and a convenient RemoteNode field to a bare network.Conn.
type Conn struct {
	network.Conn

	ID         string // ID is usually the remoteNode identifier.
	RemoteNode *node.Node
}

// Network describes a booster network: a local node, connected to other booster nodes
// using network.Conn as connector.
type Network struct {
	LocalNode *node.Node
	Conns     []*Conn
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
