/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package network provides a protable interface for network I/O, constraining
// its usage allowing only to send & receive data in a packet format.
package network

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/danielmorandini/booster/network/packet"
)

// Conn manages the serialization and deserialization of the entire
// communication system between booster nodes. Only one consumer
// per time is allowed.
type Conn struct {
	conn   net.Conn
	config Config

	// Err is filled when the connection gets closed.
	Err error

	running bool

	mutex sync.Mutex
	pe    *packet.Encoder
	pd    *packet.Decoder
}

type Config struct {
	packet.TagSet
	MaxIdle time.Duration
}

// Open creates a new connection backed by conn configured with config.
func Open(conn net.Conn, config Config) *Conn {
	return &Conn{
		conn:   conn,
		config: config,
		pe:     packet.NewEncoder(conn, config.TagSet),
		pd:     packet.NewDecoder(conn, config.TagSet),
	}
}

// RemoteAddr returns the address of the remote endpoint.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
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

// Recv reads one packet from the connection. Returns an error if the connection
// is already consuming packets.
func (c *Conn) Recv() (*packet.Packet, error) {
	if c.running {
		return nil, errors.New("conn: already consuming messages")
	}

	c.running = true
	defer func() {
		c.running = false
	}()

	p := packet.New()
	err := c.pd.Decode(p)
	if err != nil {
		return nil, err
	}

	return p, nil
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
	config Config
	l      net.Listener
}

// Listen announces to the local network address.
func Listen(network, addr string, config Config) (*Listener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	return &Listener{
		l:      l,
		config: config,
	}, nil
}

// Accept accepts incoming network connections, wrapping it into a
// booster connection.
func (l *Listener) Accept() (*Conn, error) {
	conn, err := l.l.Accept()
	if err != nil {
		return nil, err
	}

	return Open(conn, l.config), nil
}

// Close closes the underlying listener, macking Accecpt to quit
// and refute any other network connection.
func (l *Listener) Close() error {
	return l.l.Close()
}

// Dialer wraps a network dialer.
type Dialer struct {
	config Config
	d      *net.Dialer
}

// NewDialer returns a new dialer instance.
func NewDialer(d *net.Dialer, config Config) *Dialer {
	return &Dialer{
		d:      d,
		config: config,
	}
}

// DialContext dials a new net.Conn with addr, wrapping the new connection
// with a packet encoder/decoder.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (*Conn, error) {
	conn, err := d.d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	return Open(conn, d.config), nil
}

// NetworkIO implements the io.CopyN method, keeping track of the
// bandwidth while doing so.
type NetworkIO struct {
	Ticker        *time.Ticker
	NextCopyDelay time.Duration

	sync.Mutex
	N         int64 // N is the number of bytes transmitted
	Bandwidth int64
	t         int // t is the number of times CopyN was called
}

// TickerFunc calls f repeatedly after d.
// Badwidth is calculated right before calling f.
func (b *NetworkIO) TickerFunc(d time.Duration, f func()) {
	b.Lock()
	prev := b.N // prev is the previous N collected
	b.Unlock()

	b.Ticker = time.NewTicker(d)
	go func() {
		for _ = range b.Ticker.C {
			b.Lock()
			t := b.t
			N := b.N
			b.Unlock()

			if t == 0 {
				// simply return if CopyN was never called yet
				continue
			}

			// Bandwidth is populated with the number of bytes transmitted
			// since the last check
			bw := N - prev

			// Update prev
			prev += bw

			b.Lock()
			b.Bandwidth = int64(bw)
			b.Unlock()

			f()
		}
	}()
}

// CopyN copies data from src into dst, using a buffer of size n. Keeps track of
// the number of bytes copied.
func (b *NetworkIO) CopyN(dst io.Writer, src io.Reader, n int64) (int64, error) {
	n, err := io.CopyN(dst, src, n)

	b.Lock()
	b.N += n
	b.t++
	b.Unlock()

	<-time.After(b.NextCopyDelay)

	return n, err
}
