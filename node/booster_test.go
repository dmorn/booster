package node_test

import (
	"context"
	"net"
	"time"
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

type connectDialer struct {
	net.Conn
}

func (c *connectDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return c, nil
}
