package booster_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/network"
)

type conn struct {
	server     net.Conn
	client     net.Conn
	remoteAddr net.Addr
}

type addr struct {
}

func (a addr) Network() string {
	return "tcp"
}

func (a addr) String() string {
	return "localhost:1234"
}

func newConn() *conn {
	conn := new(conn)
	client, server := net.Pipe()
	conn.client = client
	conn.server = server
	conn.remoteAddr = addr{}

	return conn
}

// protocol stubs
func (c *conn) Close() error                       { return nil }
func (c *conn) LocalAddr() net.Addr                { return nil }
func (c *conn) RemoteAddr() net.Addr               { return c.remoteAddr }
func (c *conn) SetDeadline(t time.Time) error      { return nil }
func (c *conn) SetReadDeadline(t time.Time) error  { return nil }
func (c *conn) SetWriteDeadline(t time.Time) error { return nil }

func TestSendRecvHello(t *testing.T) {
	b, _ := booster.New(1234, 2345)
	ctx := context.TODO()
	conn := newConn()
	server := network.Open(conn.server, b.Netconfig)
	client := network.Open(conn.client, b.Netconfig)

	go func() {
		if err := b.SendHello(ctx, server); err != nil {
			t.Fatal(err)
		}
	}()

	_, err := b.RecvHello(ctx, client)
	if err != nil {
		t.Fatal(err)
	}
}
