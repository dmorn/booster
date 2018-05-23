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
	"fmt"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/network/packet"
)

// SendHello composes and sends an hello packet trough conn.
func (b *Booster) SendHello(ctx context.Context, conn SendCloser) error {
	log.Info.Println("booster: -> hello")

	n := b.Net().LocalNode
	pp := n.PPort()
	bp := n.BPort()

	// compose the packet
	pl := protocol.PayloadHello{
		BPort: bp,
		PPort: pp,
	}
	msg := protocol.MessageHello

	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		return err
	}

	// send
	return conn.Send(p)
}

// Ctrl commands addr to perform operation op.
func (b *Booster) Ctrl(ctx context.Context, network, addr string, op protocol.Operation) error {
	log.Info.Printf("booster: -> ctrl: %v", addr)

	conn, err := b.DialContext(ctx, network, addr)
	if err != nil {
		return fmt.Errorf("booster: unable to connect to (%v): %v", addr, err)
	}
	defer conn.Close()

	// compose the packet
	pl := protocol.PayloadCtrl{
		Operation: op,
	}
	msg := protocol.MessageCtrl

	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		return err
	}

	// send it
	if err := conn.Send(p); err != nil {
		return err
	}

	// TODO(daniel): need a response of some sort
	return nil
}

// Connect dials with laddr, creates a connect packet using raddr and tells laddr
// to connect with raddr. Both laddr and raddr are expected to point to a booster node.
//
// Closes the connection when done.
func (b *Booster) Connect(ctx context.Context, network, laddr, raddr string) (string, error) {
	log.Info.Printf("booster: -> connect: %v", raddr)

	conn, err := b.DialContext(ctx, network, laddr)
	if err != nil {
		return "", fmt.Errorf("booster: unable to connect to (%v): %v", laddr, err)
	}
	defer conn.Close()

	// compose the packet
	pl := protocol.PayloadConnect{
		Target: raddr,
	}
	msg := protocol.MessageConnect

	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		return "", err
	}

	// send it
	if err := conn.Send(p); err != nil {
		return "", err
	}

	// wait for the node packet to come and return its id
	p, err = conn.Recv()
	if err != nil {
		return "", err
	}

	m := protocol.ModulePayload
	node := new(protocol.PayloadNode)
	f := protocol.PayloadDecoders[protocol.MessageNodeStatus]

	if err = b.Net().Decode(p, m, &node, f); err != nil {
		return "", err
	}

	return node.ID, nil
}

// Disconnect dials a booster connection with addr, expecting it to be a booster node.
// It then creates a disconnect packet, telling the node to disconnect from id.
//
// Closes the connection when done.
func (b *Booster) Disconnect(ctx context.Context, network, addr, id string) error {
	log.Info.Printf("booster: -> disconnect: %v", id)

	conn, err := b.DialContext(ctx, network, addr)
	if err != nil {
		return fmt.Errorf("booster: unable to connect to (%v): %v", addr, err)
	}
	defer conn.Close()

	// compose the packet
	pl := protocol.PayloadDisconnect{
		ID: id,
	}
	msg := protocol.MessageDisconnect

	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		return err
	}

	// send it
	if err := conn.Send(p); err != nil {
		return err
	}

	// TODO(daniel): here we reuse the same reponse as for a connect request.
	// This is not actually very appropriate is it?
	p, err = conn.Recv()
	if err != nil {
		return err
	}

	m := protocol.ModulePayload
	node := new(protocol.PayloadNode)
	f := protocol.PayloadDecoders[protocol.MessageNodeStatus]
	return b.Net().Decode(p, m, &node, f)
}

type Inspection struct {
	Feature protocol.Message
	Run     func(m packet.Module) error
	PostRun func(err error)
}

func (b *Booster) Monitor(ctx context.Context, network, addr string, cmd Inspection) error {
	log.Info.Printf("booster: -> monitor: %v", addr)

	conn, err := b.DialContext(ctx, network, addr)
	if err != nil {
		return fmt.Errorf("booster: unable to connect to (%v): %v", addr, err)
	}

	// compose & send the inspect packet
	pl := protocol.PayloadMonitor{
		Features: []protocol.Message{cmd.Feature},
	}
	msg := protocol.MessageMonitor
	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		conn.Close()
		return err
	}
	if err = conn.Send(p); err != nil {
		conn.Close()
		return err
	}

	go func() {
		fail := func(err error) {
			conn.Close()

			if f := cmd.PostRun; f != nil {
				f(err)
			}
		}

		pkts, err := conn.Consume()
		if err != nil {
			fail(err)
			return
		}

		for {
			select {
			case <-ctx.Done():
				err := ctx.Err()
				fail(err)
				return
			case p = <-pkts:
				if err != nil {
					fail(err)
					return
				}

				// extract header
				h := new(protocol.Header)
				m := protocol.ModuleHeader
				f := protocol.HeaderDecoder
				if err := b.Net().Decode(p, m, &h, f); err != nil {
					fail(err)
					return
				}

				// return payload
				module, err := p.Module(string(protocol.ModulePayload))
				if err != nil {
					fail(err)
					return
				}

				if err = cmd.Run(*module); err != nil {
					fail(err)
					return
				}
			}
		}
	}()

	return nil
}
