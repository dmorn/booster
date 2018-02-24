package net

import (
	"net"

	"github.com/danielmorandini/booster/net/packet"
)

type Listener struct {
	l net.Listener
}

func Listen(network, addr string) (*Listener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	return &Listener{
		l: l,
	}, nil
}

func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	return &Conn{
		conn: conn,
		pe:  packet.NewEncoder(conn),
		pd:  packet.NewDecoder(conn),
	}, nil
}

func (l *Listener) Close() error {
	return l.l.Close()
}
