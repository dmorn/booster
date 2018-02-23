package proto_test

import (
	"testing"

	"github.com/danielmorandini/booster/proto"
	"github.com/danielmorandini/booster/node"
)

func TestGetNodes(t *testing.T) {
	b := proto.NewBoosterDefault()
	nodes := b.GetNodes()

	// first, check that there are no nodes at the beginning
	if len(nodes) != 0 {
		t.Fatalf("unexpected nodes list (wanted []): %v", nodes)
	}

	n1, _ := node.New("localhost", "1111", "1111", false)
	n2, _ := node.New("localhost", "1112", "1112", false)

	b.AddNode(n1)
	b.AddNode(n2)

	nodes = b.GetNodes()
	if len(nodes) != 2 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}
}

func TestCloseNode(t *testing.T) {
	b := proto.NewBoosterDefault()
	n, _ := node.New("localhost", "1111", "1111", false)
	b.AddNode(n)

	nodes := b.GetNodes()
	if len(nodes) != 1 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}

	n1, err := b.CloseNode(n)
	if err != nil {
		t.Fatal(err)
	}

	if n1.IsActive() {
		t.Fatal("node should not be active")
	}

	// now let's check if the node in the list was actually updated
	_, err = b.GetNode(n1.ID())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoveNode(t *testing.T) {
	b := proto.NewBoosterDefault()
	n, _ := node.New("localhost", "1111", "1111", false)
	b.AddNode(n)

	nodes := b.GetNodes()
	if len(nodes) != 1 {
		t.Logf("nodes: %v", nodes)
		t.Fatalf("unexpected node list size: %v", len(nodes))
	}

	stream, _ := b.Sub(proto.TopicNodes)
	defer func() {
		b.Unsub(stream, proto.TopicNodes)
	}()

	_, err := b.RemoveNode(n)
	if err != nil {
		t.Fatal(err)
	}

	i := <-stream
	n, ok := i.(*node.Node)
	if !ok {
		t.Fatalf("unexpected value from stream: %v type %T", i, i)
	}

	if n.IsActive() == true {
		t.Fatal("node not properly closed")
	}
}
