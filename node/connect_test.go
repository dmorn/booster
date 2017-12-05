package node_test

import (
	"testing"
	"github.com/danielmorandini/booster-network/node"
)

func TestConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping connect test in short mode")
	}

	bl := node.BOOSTER() // local booster instance
	br := node.BOOSTER() // remote booster intance

	br.Start(1090, 1091)

}
