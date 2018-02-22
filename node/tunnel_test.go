package node_test

import (
	"testing"

	"github.com/danielmorandini/booster-network/node"
)

func TestNewTunnel(t *testing.T) {
	addr := new(addr)
	tn := node.NewTunnel(addr)

	if tn.Target.String() != addr.String() {
		t.Fatalf("%v, wanted %v", tn.Target, addr)
	}

	id := "06af9eb86ad919f40ad57890edcd0977bcd96752"
	if tn.ID() != id {
		t.Fatalf("%v, wanted %v", tn.ID(), id)
	}
}
