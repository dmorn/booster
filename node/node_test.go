package node_test

import (
	"testing"
	"time"

	"github.com/danielmorandini/booster-network/node"
)

type addr struct {
}

func (a *addr) String() string {
	return "localhost:4884"
}

func (a *addr) Network() string {
	return "tcp"
}

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

	addr := new(addr)
	id := n.AddTunnel(addr)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	id1 := n.AddTunnel(addr)

	if id != id1 {
		t.Fatalf("%v, wanted %v", id, id1)
	}

	if n.Workload() != 2 {
		t.Fatalf("workload: %v, wanted 2", n.Workload())
	}
}

func TestRemoveTunnel(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	addr := new(addr)
	id := n.AddTunnel(addr)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	id1 := n.AddTunnel(addr)

	if id != id1 {
		t.Fatalf("%v, wanted %v", id, id1)
	}

	if n.Workload() != 2 {
		t.Fatalf("workload: %v, wanted 2", n.Workload())
	}

	if err := n.RemoveTunnel(id); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	if err := n.RemoveTunnel(id); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("workload: %v, wanted 0", n.Workload())
	}

	if err := n.RemoveTunnel(id); err == nil {
		t.Fatal("err should not be nil")
	}
}

func TestAck(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	addr := new(addr)
	id := n.AddTunnel(addr)

	if n.Workload() != 1 {
		t.Fatalf("workload: %v, wanted 1", n.Workload())
	}

	tn, err := n.Tunnel(id)
	if err != nil {
		t.Fatal(err)
	}

	if tn.Acks() != 0 {
		t.Fatalf("%v, wanted %v", tn.Acks(), 0)
	}

	if err = n.Ack(id); err != nil {
		t.Fatal(err)
	}

	if tn.Acks() != 1 {
		t.Fatalf("%v, wanted %v", tn.Acks(), 1)
	}

	if err := n.RemoveTunnel(id); err != nil {
		t.Fatal(err)
	}

	if n.Workload() != 0 {
		t.Fatalf("workload: %v, wanted 0", n.Workload())
	}
}

func TestStop(t *testing.T) {
	n, err := node.New("localhost", "1080", "4884", true)
	if err != nil {
		t.Fatal(err)
	}

	wait := make(chan struct{})

	go func() {
		<-n.Stop()
		wait <- struct{}{}
	}()

	if err := n.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-wait:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	}
}
