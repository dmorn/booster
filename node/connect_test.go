package node_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/danielmorandini/booster-network/node"
)

type dialer struct {
	conn node.Conn
}

func (d *dialer) DialContext(ctx context.Context, network, addr string) (node.Conn, error) {
	return d.conn, nil
}

func TestConnect(t *testing.T) {
	b := new(node.Booster)
	d := new(dialer)
	conn := new(conn)
	d.conn = conn
	b.Dialer = d
	ctx := context.Background()

	id, err := b.Connect(ctx, "tcp", "test", "node:4884")


}
