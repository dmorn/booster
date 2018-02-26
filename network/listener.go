package network

import (
	"net"

	"github.com/danielmorandini/booster/network/packet"
)

// Listener wraps a net.Listener.
type Listener struct {
	l net.Listener
}

// Listen announces to the local network address.
func Listen(network, addr string) (*Listener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	return &Listener{
		l: l,
	}, nil
}

// Accept accepts incoming network connections, wrapping it into a
// booster connection.
func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	return Open(conn, packet.NewEncoder(conn), packet.NewDecoder(conn)), nil
}

// Close closes the underlying listener, macking Accecpt to quit
// and refute any other network connection.
func (l *Listener) Close() error {
	return l.l.Close()
}
