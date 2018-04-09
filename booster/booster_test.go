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
