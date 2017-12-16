package node

import (
	"context"
	"net"
)

func (b *Booster) Status(ctx context.Context, network, laddr string) ([]*RemoteNode, error) {
	return nil, nil
}

func (b *Booster) handleStatus(ctx context.Context, conn net.Conn) error {
	return nil
}
