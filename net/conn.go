package net

import (
	"context"
	"errors"
	"io"
	"net"
	"fmt"

	"github.com/danielmorandini/booster/net/packet"
)

type Conn struct {
	Err error

	conn    io.ReadWriteCloser
	running bool

	pe   *packet.Encoder
	pd   *packet.Decoder
}

func NewConn(conn io.ReadWriteCloser, pe *packet.Encoder, pd *packet.Decoder) *Conn {
	return &Conn{
		conn: conn,
		pe: pe,
		pd: pd,
	}
}

func (c *Conn) Consume() (<-chan *packet.Packet, error) {
	if c.running {
		return nil, errors.New("conn: already running")
	}

	c.running = true
	ch := make(chan *packet.Packet)

	defer func() {
		c.running = false
	}()

	go func() {
		defer close(ch)
		for {
			p := packet.New()

			fmt.Println("connectionn is decoding...")
			err := c.pd.Decode(p)
			if err != nil {
				c.Err = err
				return
			}
			fmt.Println("connectio decoded.")

			ch <- p
		}
	}()

	return ch, nil
}

func (c *Conn) Send(p *packet.Packet) error {
	return c.pe.Encode(p)
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
		return nil, err
	}

	return &Conn{
		conn: conn,
		pe:  packet.NewEncoder(conn),
		pd:  packet.NewDecoder(conn),
	}, nil
}
