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

package booster

import (
	"context"
	"net"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
)

// RecvHello takes a raw network connection and reads the next message coming. It is expected
// to be an hello message, introducing a new remote booster instance.
//
// With the informations contained in the packet, it creates a new booster connection
// and returns it.
func (b *Booster) RecvHello(ctx context.Context, conn *network.Conn) (*Conn, error) {
	fail := func(err error) (*Conn, error) {
		conn.Close()
		return nil, err
	}

	// Read the hello packet
	p, err := conn.Recv()
	if err != nil {
		return fail(err)
	}

	m := protocol.ModulePayload
	pl := new(protocol.PayloadHello)
	if err := b.Net().Decode(p, m, &pl); err != nil {
		return fail(err)
	}

	// extract node information from the message
	pp := pl.PPort
	bp := pl.BPort
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	log.Info.Printf("booster: <- hello: %v %v-%v", host, pp, bp)

	// create new node with the information collected
	node, err := node.New(host, pp, bp, false)
	if err != nil {
		return fail(err)
	}

	return b.Net().NewConn(conn, node, node.ID()), nil
}
