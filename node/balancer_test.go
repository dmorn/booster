package node_test

import (
	"testing"

	"github.com/danielmorandini/booster-network/node"
)

func TestGetNodes(t *testing.T) {
	b := node.NewBoosterDefault()
	nodes := b.GetNodes()

	// fisrt check that there are no nodes at the beginning
	if len(nodes) != 0 {
		t.Fatalf("unexpected nodes list (wanted []): %v", nodes)
	}

	n1 := node.NewRemoteNode("host", "port1", "bport")
	n2 := node.NewRemoteNode("host", "port2", "bport")

	b.AddNode(n1)
	b.AddNode(n2)

	nodes = b.GetNodes()
	if len(nodes) != 2 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}
}
