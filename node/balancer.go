package node

import (
	"errors"
	"log"
)

// Balancer is a LoadBalancer implementation. It manages nodes, providing
// functionalities to store and manage a set of Nodes. Uses PubSub
// as a notification mechanism to let others know which operations are
// performed.
// (check inspect.go for an example)
type Balancer struct {
	*log.Logger
	PubSub

	// rootNode is the root node of every other remote one. Its workload
	// is the sum of the workloads of the nodes plus its own.
	rootNode *Node
	nodes    map[string]*Node
}

// NewBalancer returns a new balancer instance.
func NewBalancer(log *log.Logger, ps PubSub) *Balancer {
	b := new(Balancer)
	b.Logger = log
	b.PubSub = ps
	b.nodes = make(map[string]*Node)

	return b
}

// GetNodeBalanced collects the workload of its registered nodes,
// and compares them to the workload of the root node.
//
// Returns an error if no candidate is found, either because
// none was provided or because no entry's workload was under
// the treshold.
//
// exp is a list of ids, which are considered as nodes that should
// not be taken into consideration.
func (b *Balancer) GetNodeBalanced(exp ...string) (*Node, error) {
	if b.rootNode == nil {
		return nil, errors.New("balancer: found nil rootNode")
	}

	b.rootNode.Lock()
	tr := b.rootNode.workload
	b.rootNode.Unlock()

	var c *Node // candidate entry
	var twl int // total workload

	for _, e := range b.nodes {
		// do not condider non active nodes
		if !e.IsActive {
			continue
		}

		e.Lock()
		ewl := e.workload
		twl += ewl
		e.Unlock()

		// check if node is in the exceptions
		if isIn(e.ID(), exp...) {
			continue
		}

		if c == nil {
			c = e
		}

		c.Lock()
		cwl := c.workload // candidate workload
		c.Unlock()

		if ewl < cwl {
			c = e
		}
	}

	// we did not find any suitable node
	if c == nil {
		if isIn(b.rootNode.ID(), exp...) {
			return nil, errors.New("balancer: no suitable node found")
		}

		return b.rootNode, nil
	}

	// tr is the sum of the local workload and the remote node's workload.
	// this is why we have to subtract the total remote workload to understand
	// how the load on this node is.
	if c.workload > (tr - twl) {
		// return the candidate even if the local node is the most suitable one
		if isIn(b.rootNode.ID(), exp...) {
			return c, nil
		}

		return b.rootNode, nil
	}

	return c, nil
}

func isIn(id string, ids ...string) bool {
	for _, v := range ids {
		if id == v {
			return true
		}
	}
	return false
}

// SetRootNode sets the rootNode of the balancer. Be careful that this value HAS to be set before using the
// balancer.
func (b *Balancer) SetRootNode(node *Node) {
	node.Lock()
	node.lastOperation.op = BoosterNodeAdded
	node.Unlock()

	b.rootNode = node
}

// GetNode returns the node associated with id.
// Returns an error if no node with this id is found.
func (b *Balancer) GetNode(id string) (*Node, error) {
	if e, ok := b.nodes[id]; ok {
		return e, nil
	}

	return nil, errors.New("balancer: " + id + " not found")
}

// GetNodes returns all stored nodes.
func (b *Balancer) GetNodes() []*Node {
	nodes := []*Node{}
	if b.rootNode != nil {
		nodes = append(nodes, b.rootNode)
	}

	for _, val := range b.nodes {
		nodes = append(nodes, val)
	}

	return nodes
}

// UpdateNode updates the workload of a node. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) UpdateNode(node *Node, workload int, target string) (*Node, error) {
	node.Lock()
	node.IsActive = true
	node.workload = workload
	node.lastOperation.op = BoosterNodeUpdated
	node.lastOperation.id = target
	node.Unlock()

	b.Pub(node, TopicNodes)
	return node, nil
}

// AddNode adds a new entry to the monitored nodes. If a node with the same
// id is already present, it removes it. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) AddNode(node *Node) (*Node, error) {
	if _, ok := b.nodes[node.ID()]; ok {
		// close, remove it and substitute
		b.CloseNode(node.ID())
		b.RemoveNode(node.ID())
	}

	node.Lock()
	defer node.Unlock()

	b.Printf("balancer: adding node (%v)", node.ID())
	node.lastOperation.op = BoosterNodeAdded
	b.nodes[node.ID()] = node
	b.Pub(node, TopicNodes)

	return node, nil
}

// CloseNode calls Close on the node with id. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) CloseNode(id string) (*Node, error) {
	node, err := b.GetNode(id)
	if err != nil {
		return nil, err
	}

	node.Lock()
	defer node.Unlock()

	b.Printf("balancer: closing node (%v)\n", id)
	lastOp := node.lastOperation.op
	if lastOp == BoosterNodeClosed {
		return nil, errors.New("balancer: node (" + node.ID() + ") already closed")
	}

	node.Close()
	b.Pub(node, TopicNodes)

	return node, nil
}

// RemoveNode removes the entry labeled with id.
// Returns false if no entry was found. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) RemoveNode(id string) (*Node, error) {
	node, err := b.GetNode(id)
	if err != nil {
		return nil, err
	}

	node.Lock()
	defer node.Unlock()

	b.Printf("balancer: removing node (%v)\n", id)
	lastOp := node.lastOperation.op
	if lastOp == BoosterNodeRemoved {
		return nil, errors.New("balancer: node (" + node.ID() + ") already removed")
	}

	node.lastOperation.op = BoosterNodeRemoved
	delete(b.nodes, id)
	b.Pub(node, TopicNodes)

	return node, nil
}
