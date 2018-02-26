package network

import (
	"context"
	"net"

	"github.com/danielmorandini/booster/network/packet"
)

// Dialer wraps a network dialer.
type Dialer struct {
	d net.Dialer
}

// DialContext dials a new booster connection, starting the heartbeat procedure on it.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (*Conn, error) {
	conn, err := d.d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn: conn,
		pe:   packet.NewEncoder(conn),
		pd:   packet.NewDecoder(conn),
	}, nil
}
