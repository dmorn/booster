package node_test

import(
	"testing"
	"context"

	"github.com/danielmorandini/booster-network/node"
)

func TestHello(t *testing.T) {
	mockConn := newConn()
	d := new(connectDialer)
	d.Conn = mockConn.client

	br := node.BOOSTER()
	br.Dialer = d

	trgAddr := "127.0.1.65:1090" // expected address returned by Hello

	c := make(chan error)
	go func() {
		ctx := context.Background()
		_, paddr, err := br.Hello(ctx, "fakenet", "fakeaddr")
	}
}
