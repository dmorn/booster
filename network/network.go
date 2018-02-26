package network

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/danielmorandini/booster/network/packet"
)

// Conn manages the serialization and deserialization of the entire
// communication system between booster nodes. Only one consumer
// per time is allowed.
type Conn struct {
	// Err is filled when the connection gets closed.
	Err error

	conn    io.ReadWriteCloser
	running bool

	mutex sync.Mutex
	pe    *packet.Encoder
	pd    *packet.Decoder
}

// Open creates a new Conn. Used mainly for testing outside of the package.
// Usally connections are created using the listener.
func Open(conn io.ReadWriteCloser, pe *packet.Encoder, pd *packet.Decoder) *Conn {
	return &Conn{
		conn: conn,
		pe:   pe,
		pd:   pd,
	}
}

// Consume keeps on reading on the connection, decoding each message received and
// exiting with an error if it is not able to decode the data collected into a
// packet.
// Each packet is sent into the decoded channel. When it gets closed, check
// c.Err.
func (c *Conn) Consume() (<-chan *packet.Packet, error) {
	if c.running {
		return nil, errors.New("conn: already running")
	}

	c.running = true
	defer func() {
		c.running = false
	}()

	ch := make(chan *packet.Packet)
	go func() {
		defer close(ch)
		for {
			p := packet.New()

			err := c.pd.Decode(p)
			if err != nil {
				c.Err = err
				return
			}

			ch <- p
		}
	}()

	return ch, nil
}

// Send sends the packet trough the connection. It is safe to call from multiple
// goroutines.
func (c *Conn) Send(p *packet.Packet) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.pe.Encode(p)
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}

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
