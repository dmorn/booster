package booster

import (
	"errors"
	"log"
	"sync"
	"net"

	"github.com/danielmorandini/booster-network/tracer"
)

type Node interface {
	tracer.Pinger
	BinaryEncoder

	Workload() int
	IsActive() bool
	SetIsActive(bool)
	IsLocal() bool
	ProxyAddr() net.Addr
	Ack(net.Addr) error
	AddTunnel(net.Addr)
	RemoveTunnel(net.Addr) error
	Close() error
	Stop() chan struct{}
}

// Balancer is a LoadBalancer implementation. It manages nodes, providing
// functionalities to store and manage a set of Nodes. Uses PubSub
// as a notification mechanism to let others know which operations are
// performed.
// (check inspect.go for an example)
type Balancer struct {
	*log.Logger
	PubSub

	sync.Mutex
	// rootNode is the root node of every other remote one. Its workload
	// is the sum of the workloads of the nodes plus its own.
	rootNode Node
	nodes    map[string]Node
}

// NewBalancer returns a new balancer instance.
func NewBalancer(log *log.Logger, ps PubSub) *Balancer {
	b := new(Balancer)
	b.Logger = log
	b.PubSub = ps
	b.nodes = make(map[string]Node)

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
func (b *Balancer) GetNodeBalanced(exp ...string) (Node, error) {
	if b.RootNode() == nil {
		return nil, errors.New("balancer: found nil rootNode")
	}

	tr := b.RootNode().Workload()

	var c Node // candidate entry
	var twl int // total workload

	b.Lock()
	for _, e := range b.nodes {
		// do not condider non active nodes
		if !e.IsActive() {
			continue
		}

		ewl := e.Workload()
		twl += ewl

		// check if node is in the exceptions
		if isIn(e.ID(), exp...) {
			continue
		}

		if c == nil {
			c = e
		}

		cwl := c.Workload() // candidate workload

		if ewl < cwl {
			c = e
		}
	}
	b.Unlock()

	// we did not find any suitable node
	if c == nil {
		if isIn(b.RootNode().ID(), exp...) {
			return nil, errors.New("balancer: no suitable node found")
		}

		return b.RootNode(), nil
	}

	// tr is the sum of the local workload and the remote node's workload.
	// this is why we have to subtract the total remote workload to understand
	// how the load on this node is.
	if c.Workload() > (tr - twl) {
		// return the candidate even if the local node is the most suitable one
		if isIn(b.RootNode().ID(), exp...) {
			return c, nil
		}

		return b.RootNode(), nil
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
func (b *Balancer) SetRootNode(node Node) {
	b.Lock()
	defer b.Unlock()

	b.rootNode = node
}

func (b *Balancer) RootNode() Node {
	b.Lock()
	defer b.Unlock()

	return b.rootNode
}

// GetNode returns the node associated with id.
// Returns an error if no node with this id is found.
func (b *Balancer) GetNode(id string) (Node, error) {
	b.Lock()
	defer b.Unlock()

	if e, ok := b.nodes[id]; ok {
		return e, nil
	}

	return nil, errors.New("balancer: " + id + " not found")
}

// GetNodes returns all stored nodes.
func (b *Balancer) GetNodes() []Node {
	nodes := []Node{}
	if b.RootNode() != nil {
		nodes = append(nodes, b.RootNode())
	}

	b.Lock()
	defer b.Unlock()
	for _, val := range b.nodes {
		nodes = append(nodes, val)
	}

	return nodes
}

// AddNode adds a new entry to the monitored nodes. If a node with the same
// id is already present, it removes it. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) AddNode(node Node) (Node, error) {
	if _, ok := b.nodes[node.ID()]; ok {
		return nil, errors.New("balancer: node (" + node.ID() + ") already present")
	}

	b.Printf("balancer: adding node (%v)", node.ID())
	b.nodes[node.ID()] = node
	b.Pub(node, TopicNodes)

	return node, nil
}

func (b *Balancer) Ack(node Node, target net.Addr) error {
	b.Printf("balancer: acknoledging (%v) on node (%v)", target, node.ID())

	if err := node.Ack(target); err != nil {
		return err
	}

	b.Pub(node, TopicNodes)
	return nil
}

func (b *Balancer) RemoveTunnel(node Node, target net.Addr) error {
	b.Printf("balancer: removing (%v) on node (%v)", target, node.ID())

	if err := node.RemoveTunnel(target); err != nil {
		return err
	}

	b.Pub(node, TopicNodes)
	return nil
}

func (b *Balancer) AddTunnel(node Node, target net.Addr) {
	b.Printf("balancer: adding tunnel (%v) to node (%v)", target, node.ID())

	node.AddTunnel(target)
	b.Pub(node, TopicNodes)
}

// CloseNode calls Close on the node with id. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) CloseNode(node Node) (Node, error) {
	b.Printf("balancer: closing node (%v)", node.ID())

	node.Close()
	b.Pub(node, TopicNodes)

	return node, nil
}

// RemoveNode removes the entry labeled with id.
// Returns false if no entry was found. Publishes the updated node to the pubsub
// with topic TopicNodes.
func (b *Balancer) RemoveNode(node Node) (Node, error) {
	b.Printf("balancer: removing node (%v)\n", node.ID())

	delete(b.nodes, node.ID())
	b.Pub(node, TopicNodes)

	return node, nil
}
