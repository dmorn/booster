package node

import (
	"errors"
	"log"
	"net"

	"github.com/danielmorandini/booster-network/pubsub"
)

// Balancer is a LoadBalancer implementation. It manages nodes, providing
// functionalities to store and manage a set of RemoteNodes. Uses PubSub
// as a notification mechanism to let others know which operations are
// performed.
// (check inspect.go for an example)
type Balancer struct {
	*log.Logger
	*pubsub.PubSub

	nodes map[string]*RemoteNode
}

// NewBalancer returns a new balancer instance.
func NewBalancer(log *log.Logger, ps *pubsub.PubSub) *Balancer {
	b := new(Balancer)
	b.Logger = log
	b.PubSub = ps
	b.nodes = make(map[string]*RemoteNode)

	return b
}

// GetNodeBalanced collects the workload of its registered nodes,
// and compares them to the tr workload, that represents the
// workload that the remote node is supposed to "beat" in order
// to be used next.
//
// Returns an error if no candidate is found, either because
// none was provided or because no entry's workload was under
// the treshold.
func (b *Balancer) GetNodeBalanced(tr int) (*RemoteNode, error) {
	var c *RemoteNode // candidate entry
	var twl int       // total workload

	for _, e := range b.nodes {
		// do not condider non active nodes
		if !e.IsActive {
			continue
		}

		if c == nil {
			c = e
		}

		e.Lock()
		ewl := e.workload
		twl += ewl
		e.Unlock()

		c.Lock()
		cwl := c.workload // candidate workload
		c.Unlock()

		if ewl < cwl {
			c = e
		}
	}

	if c == nil {
		return nil, errors.New("booster balancer: no remote boosters connected")
	}

	// tr is the sum of the local workload and the remote node's workload.
	// this is why we have to subtract the total remote workload to understand
	// how is the load on this node.
	if c.workload > (tr - twl) {
		return nil, errors.New("booster balancer: use local proxy")
	}

	return c, nil
}

// GetNode returns the node associated with id.
// Returns an error if no node with this id is found.
func (b *Balancer) GetNode(id string) (*RemoteNode, error) {
	if e, ok := b.nodes[id]; ok {
		return e, nil
	}

	return nil, errors.New("balancer: " + id + " not found")
}

// GetNodes returns all stored nodes.
func (b *Balancer) GetNodes() []*RemoteNode {
	nodes := []*RemoteNode{}
	for _, val := range b.nodes {
		nodes = append(nodes, val)
	}

	return nodes
}

// UpdateNode updates the workload of a node. Returns error if no
// node is found related to id. Publishes the updated node to the pubsub
// with topic TopicRemoteNodes.
func (b *Balancer) UpdateNode(id string, workload int) (*RemoteNode, error) {
	node, err := b.GetNode(id)
	if err != nil {
		return nil, err
	}

	node.Lock()
	node.workload = workload
	node.LastOperation = BoosterNodeUpdated
	node.Unlock()

	b.Pub(node, TopicRemoteNodes)
	return node, nil
}

// AddNode adds a new entry to the monitored nodes. If a node with the same
// id is already present, it removes it. Publishes the updated node to the pubsub
// with topic TopicRemoteNodes.
func (b *Balancer) AddNode(node *RemoteNode) (*RemoteNode, error) {
	if _, ok := b.nodes[node.ID]; ok {
		// close, remove it and substitute
		b.CloseNode(node.ID)
		b.RemoveNode(node.ID)
	}

	b.Printf("balancer: adding node %v (%v)", node.ID, net.JoinHostPort(node.Host, node.Pport))
	node.Lock()
	node.LastOperation = BoosterNodeAdded
	node.Unlock()
	b.nodes[node.ID] = node

	b.Pub(node, TopicRemoteNodes)
	return node, nil
}

// CloseNode calls Close on the node with id. Publishes the updated node to the pubsub
// with topic TopicRemoteNodes.
func (b *Balancer) CloseNode(id string) (*RemoteNode, error) {
	node, err := b.GetNode(id)
	if err != nil {
		return nil, err
	}

	node.Lock()
	lastOp := node.LastOperation
	node.Unlock()
	if lastOp == BoosterNodeClosed {
		return nil, errors.New("balancer: node (" + node.ID + ") already closed")
	}

	b.Printf("balancer: closing node %v\n", id)
	node.Close()

	b.Pub(node, TopicRemoteNodes)

	return node, nil
}

// RemoveNode removes the entry labeled with id.
// Returns false if no entry was found. Publishes the updated node to the pubsub
// with topic TopicRemoteNodes.
func (b *Balancer) RemoveNode(id string) (*RemoteNode, error) {
	node, err := b.GetNode(id)
	if err != nil {
		return nil, err
	}

	node.Lock()
	lastOp := node.LastOperation
	node.Unlock()
	if lastOp == BoosterNodeRemoved {
		return nil, errors.New("balancer: node (" + node.ID + ") already removed")
	}

	b.Printf("balancer: removing node %v\n", id)
	node.Lock()
	node.LastOperation = BoosterNodeRemoved
	node.Unlock()
	delete(b.nodes, id)

	b.Pub(node, TopicRemoteNodes)
	return node, nil
}
