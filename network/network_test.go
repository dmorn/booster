package network_test

import (
	"io"
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
	server io.ReadWriteCloser
	client io.ReadWriteCloser
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

func TestAcceptSend(t *testing.T) {
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
