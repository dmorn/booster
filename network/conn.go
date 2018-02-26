package network

import (
	"errors"
	"io"
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
	pe *packet.Encoder
	pd *packet.Decoder
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
