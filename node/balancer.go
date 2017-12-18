package node

import (
	"errors"
	"log"
	"net"
)

// Balancer is a LoadBalancer implementation
type Balancer struct {
	*log.Logger

	nodes map[string]*RemoteNode
}

// NewBalancer returns a new balancer instance.
func NewBalancer(log *log.Logger) *Balancer {
	b := new(Balancer)
	b.Logger = log
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

		if ewl < cwl && e.IsActive {
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
	nodes := make([]*RemoteNode, len(b.nodes))
	for _, val := range b.nodes {
		nodes = append(nodes, val)
	}

	return nodes
}

// AddNode adds a new entry to the monitored nodes. conn is expected to
// come from a booster node.
// Returns the remote node identifier.
func (b *Balancer) AddNode(node *RemoteNode) error {
	if _, ok := b.nodes[node.ID]; ok {
		// remove it and substitute
		b.RemoveNode(node.ID)
	}

	b.Printf("balancer: adding proxy %v (%v)", node.ID, net.JoinHostPort(node.Host, node.Pport))
	b.nodes[node.ID] = node

	return nil
}

// RemoveNode removes the entry labeled with id.
// Returns false if no entry was found.
func (b *Balancer) RemoveNode(id string) bool {
	if e, ok := b.nodes[id]; ok {
		b.Printf("balancer: removing proxy %v\n", id)
		e.cancel()
		delete(b.nodes, id)

		return ok
	}

	b.Printf("balancer: %v not found\n", id)
	return false
}
