package net

import (
	"net"

	"github.com/danielmorandini/booster-network/packet"
)

type Listener struct {
	l net.Listener
}

func Listen(network, addr string) (*Listener, error) {
	return net.Listen(network, addr)
}

func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn: conn,
		ped:  packet.NewEncoderDecoder(conn),
	}, nil
}

func (l *Listener) Close() error {
	return l.l.Close()
}
