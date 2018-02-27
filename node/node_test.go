package node_test

import (
	"testing"

	"github.com/danielmorandini/booster/node"
)

func TestNewNode(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("%v, wanted %v", n.Workload(), 0)
	}

	if !n.IsLocal() {
		t.Fatalf("%v, wanted true. Node is not local", n.IsLocal())
	}

	if n.IsActive() {
		t.Fatalf("node is active")
	}

	n.SetIsActive(true)
	if !n.IsActive() {
		t.Fatalf("node is NOT active")
	}
}

func TestAddTunnel(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	addr := "host:8888"
	n.AddTunnel(addr)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	n.AddTunnel(addr)

	if n.Workload() != 2 {
		t.Fatalf("workload: %v, wanted 2", n.Workload())
	}
}

func TestRemoveTunnel(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	addr := "host:8888"
	n.AddTunnel(addr)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	n.AddTunnel(addr)

	if n.Workload() != 2 {
		t.Fatalf("workload: %v, wanted 2", n.Workload())
	}

	if err := n.RemoveTunnel(addr); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	if err := n.RemoveTunnel(addr); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("workload: %v, wanted 0", n.Workload())
	}

	if err := n.RemoveTunnel(addr); err == nil {
		t.Fatal("err should not be nil")
	}
}

func TestAck(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	addr := "host:8888"
	n.AddTunnel(addr)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	tn, err := n.Tunnel(addr)
	if err != nil {
		t.Fatal(err)
	}

	if tn.Acks() != 0 {
		t.Fatalf("%v, wanted %v", tn.Acks(), 0)
	}

	if err = n.Ack(addr); err != nil {
		t.Fatal(err)
	}

	if tn.Acks() != 1 {
		t.Fatalf("%v, wanted %v", tn.Acks(), 1)
	}

	if err := n.RemoveTunnel(addr); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("workload: %v, wanted 0", n.Workload())
	}
}
