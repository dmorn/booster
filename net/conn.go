package net

import (
	"context"
	"net"

	"github.com/danielmorandini/booster-network/packet"
)

type Conn struct {
	conn net.Conn
	ped *packet.EncoderDecoder
}

func (c *Conn) Accept() (*packet.Packet, error) {
	p := packet.New()
	err := c.ped.Decode(p)

	return p, err
}

func (c *Conn) Send(p *packet.Packet) error {
	return c.ped.Encode(p)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

type Dialer struct {
	d net.Dialer
}

func (d *Dialer) DialContext(ctx context.Context, network, addr string) (*Conn, error) {
	conn, err := d.d.DialContext(ctx, network, addr)
	if err != nil {
		nil, err
	}

	return &Conn{
		conn: conn,
		ped: packet.NewEncoderDecoder(conn),
	}, nil
}
