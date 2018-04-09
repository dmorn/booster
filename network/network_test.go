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

package network_test

import (
	"net"
	"testing"
	"time"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

var netconfig network.Config = network.Config{
	TagSet: packet.TagSet{
		PacketOpeningTag:  protocol.PacketOpeningTag,
		PacketClosingTag:  protocol.PacketClosingTag,
		PayloadClosingTag: protocol.PayloadClosingTag,
		Separator:         protocol.Separator,
	},
}

type conn struct {
	server net.Conn
	client net.Conn
}

func newConn() *conn {
	conn := new(conn)
	client, server := net.Pipe()
	conn.client = client
	conn.server = server

	return conn
}

// protocol stubs
func (c *conn) Close() error { return nil }

func TestSendRecv(t *testing.T) {
	mc := newConn()
	client := network.Open(mc.client, netconfig)
	server := network.Open(mc.server, netconfig)

	p := packet.New()
	pl := []byte("hello")
	_, err := p.AddModule("fo", pl, 0)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		if err := server.Send(p); err != nil {
			t.Fatal(err)
		}
	}()

	rp, err := client.Recv()
	if err != nil {
		t.Fatal(err)
	}

	m, err := rp.Module("fo")
	if err != nil {
		t.Fatal(err)
	}

	if len(m.Payload()) != len(pl) {
		t.Fatalf("%v, wanted %v", m.Payload(), pl)
	}
}

func TestSendConsume(t *testing.T) {
	mc := newConn()
	client := network.Open(mc.client, netconfig)
	server := network.Open(mc.server, netconfig)

	c := make(chan *packet.Packet)
	go func() {
		pkts, err := server.Consume()
		if err != nil {
			t.Fatal(err)
		}

		c <- <-pkts
	}()

	p := packet.New()
	_, err := p.AddModule("fo", []byte{1}, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.AddModule("of", []byte{1}, 0)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Send(p); err != nil {
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

	case <-time.After(1 * time.Second):
		t.Fatal("timeout: couldn't read packet")
	}
}
