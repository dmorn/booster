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
	"bufio"
	"bytes"
	"net"
	"testing"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
)

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

var netconfig = booster.DefaultNetConfig

// protocol stubs
func (c *conn) Close() error { return nil }

func TestSend(t *testing.T) {
	mc := newConn()
	client := network.Open(mc.client, netconfig)

	p := packet.New()
	p.M = packet.Metadata{
		Encoding:    1,
		Compression: 2,
		Encryption:  3,
	}

	pl := []byte("hello")
	_, err := p.AddModule("fo", pl)
	if err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 0, 1024)
	errc := make(chan error)

	go func() {
		r := bufio.NewReader(mc.server)
		buf := make([]byte, 8)
		for {
			n, err := r.Read(buf)
			if err != nil {
				errc <- err
				return
			}

			p := buf[:n]
			data = append(data, p...)

			if bytes.HasSuffix(data, []byte(netconfig.TagSet.PacketClosingTag)) {
				errc <- nil
				return
			}
		}
	}()

	if err := client.Send(p); err != nil {
		t.Fatal(err)
	}

	// wait for reading to be completed
	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	exo := []byte{62, 1, 58, 2, 58, 3, 58, 1, 91, 102, 111, 58, 0, 5, 58, 104, 101, 108, 108, 111, 93, 60} // >1:2:3:1[fo:5:hello]<

	if !bytes.Equal(exo, data) {
		t.Fatalf("content of expected output and data differ: (%v) != (%v)", exo, data)
	}
}

func TestRecv(t *testing.T) {
	mc := newConn()
	server := network.Open(mc.server, netconfig)

	p := []byte{62, 1, 58, 2, 58, 3, 58, 1, 91, 102, 111, 58, 0, 5, 58, 104, 101, 108, 108, 111, 93, 60} // >1:2:3:1[fo:5:hello]<
	errc := make(chan error)

	go func() {
		_, err := mc.client.Write(p)
		errc <- err
	}()

	packet, err := server.Recv()
	if err != nil {
		t.Fatal(err)
	}

	// wait for writing to return
	if err = <-errc; err != nil {
		t.Fatal(err)
	}

	// test full packet content
	m := packet.M
	if m.Encoding != 1 {
		t.Fatalf("unexpected encoding: wanted 1, found %v", m.Encoding)
	}
	if m.Compression != 2 {
		t.Fatalf("unexpected compression: wanted 2, found %v", m.Compression)
	}
	if m.Encryption != 3 {
		t.Fatalf("unexpected encryption: wanted 3, found %v", m.Encryption)
	}

	module, err := packet.Module("fo")
	if err != nil {
		t.Fatal(err)
	}

	if string(module.Payload()) != "hello" {
		t.Fatalf("unexpected payload: wanted hello, found %v", module.Payload())
	}

}
