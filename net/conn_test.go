package net_test

import (
	"testing"
	"net"
	"time"

	bnet "github.com/danielmorandini/booster/net"
	"github.com/danielmorandini/booster/net/packet"
)

type conn struct {
	server     net.Conn
	client     net.Conn
	remoteAddr net.Addr
}

func newConn() *conn {
	conn := new(conn)
	client, server := net.Pipe()
	conn.client = client
	conn.server = server

	return conn
}

// protocol stubs
func (c *conn) Close() error                       { return nil }
func (c *conn) LocalAddr() net.Addr                { return nil }
func (c *conn) RemoteAddr() net.Addr               { return c.remoteAddr }
func (c *conn) SetDeadline(t time.Time) error      { return nil }
func (c *conn) SetReadDeadline(t time.Time) error  { return nil }
func (c *conn) SetWriteDeadline(t time.Time) error { return nil }

func TestAcceptSend(t *testing.T) {
	mc := newConn()
	conn := bnet.NewConn(mc.client, packet.NewEncoder(mc.client), packet.NewDecoder(mc.server))

	c := make(chan *packet.Packet)
	go func() {
		pkts, err := conn.Accept()
		if err != nil {
			t.Fatal(err)
		}

		c <- <- pkts
	}()

	p := packet.New()
	_, err := p.AddModule("fo", []byte{1})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Send(p); err != nil {
		t.Fatal(err)
	}

	select {
	case p1 := <-c:
		if p1 == nil {
			t.Fatalf("unexpected nil packet")
		}

		if _, err := p1.Module("fo"); err != nil {
			t.Fatalf("packet %+v: %v", p1, err)
		}

	case <-time.After(1*time.Second):
		t.Fatal("timeout: couldn't read packet")
	}
}
