package net

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/danielmorandini/booster/net/packet"
)

type Conn struct {
	Err error

	conn    net.Conn
	running bool

	mutex sync.Mutex
	ped   *packet.EncoderDecoder
}

func (c *Conn) Accept() (<-chan *packet.Packet, error) {
	if c.running {
		return nil, errors.New("conn: already running")
	}

	c.running = true
	ch := make(chan *packet.Packet)
	errc := make(chan error)

	defer func() {
		c.running = false
		close(ch)
		close(errc)
	}()

	go func() {
		for {
			p := packet.New()
			if err := c.ped.Decode(p); err != nil {
				c.Err = err
				return
			}

			ch <- p
		}
	}()

	return ch, err
}

func (c *Conn) Send(p *packet.Packet) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
		return nil, err
	}

	return &Conn{
		conn: conn,
		ped:  packet.NewEncoderDecoder(conn),
	}, nil
}
