package node_test

import (
	"testing"

	"github.com/danielmorandini/booster-network/node"
)

func TestGetNodes(t *testing.T) {
	b := node.NewBoosterDefault()
	nodes := b.GetNodes()

	// first, check that there are no nodes at the beginning
	if len(nodes) != 0 {
		t.Fatalf("unexpected nodes list (wanted []): %v", nodes)
	}

	n1, _ := node.NewNode("localhost", "1111", "1111")
	n2, _ := node.NewNode("localhost", "1112", "1112")

	b.AddNode(n1)
	b.AddNode(n2)

	nodes = b.GetNodes()
	if len(nodes) != 2 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}
}

func TestCloseNode(t *testing.T) {
	b := node.NewBoosterDefault()
	n, _ := node.NewNode("localhost", "1111", "1111")
	b.AddNode(n)

	nodes := b.GetNodes()
	if len(nodes) != 1 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}

	n1, err := b.CloseNode(n.ID())
	if err != nil {
		t.Fatal(err)
	}

	if n1.IsActive {
		t.Fatal("node should not be active")
	}

	if n1.LastOperation() != node.BoosterNodeClosed {
		t.Fatalf("unexpected node last operation: found %v, wanted %v", n.LastOperation(), node.BoosterNodeClosed)
	}

	// now let's check if the node in the list was actually updated
	n, err = b.GetNode(n1.ID())
	if err != nil {
		t.Fatal(err)
	}

	if n.LastOperation() != node.BoosterNodeClosed {
		t.Fatalf("unexpected node last operation in nodes list: found %v, wanted %v", n.LastOperation(), node.BoosterNodeClosed)
	}
}

func TestRemoveNode(t *testing.T) {
	b := node.NewBoosterDefault()
	n, _ := node.NewNode("localhost", "1111", "1111")
	b.AddNode(n)

	nodes := b.GetNodes()
	if len(nodes) != 1 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}

	stream, _ := b.Sub(node.TopicNodes)
	defer func() {
		b.Unsub(stream, node.TopicNodes)
	}()

	n, err := b.RemoveNode(n.ID())
	if err != nil {
		t.Fatal(err)
	}

	i := <-stream
	n, ok := i.(*node.Node)
	if !ok {
		t.Fatalf("unexpected value from stream: %v type %T", i, i)
	}

	if n.IsActive == true {
		t.Fatal("node not properly closed")
	}

	if n.LastOperation() != node.BoosterNodeRemoved {
		t.Fatalf("unexpected node last operation: found %v, wanted %v", n.LastOperation(), node.BoosterNodeRemoved)
	}
}
