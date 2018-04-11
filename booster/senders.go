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
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

// SendHello composes and sends an hello packet trough conn.
func (b *Booster) SendHello(ctx context.Context, conn SendCloser) error {
	log.Info.Println("booster: -> hello")

	// create the modules
	h, err := protocol.HelloHeader()
	if err != nil {
		return err
	}

	n := Nets.Get(b.ID).LocalNode
	pp := n.PPort()
	bp := n.BPort()

	pl, err := protocol.EncodePayloadHello(bp, pp)
	if err != nil {
		return err
	}

	// compose the packet
	p := packet.New()
	enc := protocol.EncodingProtobuf
	if _, err = p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return err
	}
	if _, err = p.AddModule(protocol.ModulePayload, pl, enc); err != nil {
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
	h, err := protocol.CtrlHeader()
	if err != nil {
		return err
	}
	pl, err := protocol.EncodePayloadCtrl(op)
	if err != nil {
		return err
	}

	p := packet.New()
	enc := protocol.EncodingProtobuf
	if _, err := p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return err
	}
	if _, err := p.AddModule(protocol.ModulePayload, pl, enc); err != nil {
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
	h, err := protocol.ConnectHeader()
	if err != nil {
		return "", err
	}
	pl, err := protocol.EncodePayloadConnect(raddr)
	if err != nil {
		return "", err
	}

	p := packet.New()
	enc := protocol.EncodingProtobuf
	if _, err := p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return "", err
	}
	if _, err := p.AddModule(protocol.ModulePayload, pl, enc); err != nil {
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

	if err = ValidatePacket(p); err != nil {
		return "", err
	}

	praw, err := p.Module(protocol.ModulePayload)
	if err != nil {
		return "", err
	}

	node, err := protocol.DecodePayloadNode(praw.Payload())
	if err != nil {
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
	h, err := protocol.DisconnectHeader()
	if err != nil {
		return err
	}

	pl, err := protocol.EncodePayloadDisconnect(id)
	if err != nil {
		return err
	}

	p := packet.New()
	enc := protocol.EncodingProtobuf
	if _, err := p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return err
	}
	if _, err := p.AddModule(protocol.ModulePayload, pl, enc); err != nil {
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

	if err = ValidatePacket(p); err != nil {
		return err
	}

	praw, err := p.Module(protocol.ModulePayload)
	if err != nil {
		return err
	}
	_, err = protocol.DecodePayloadNode(praw.Payload())
	if err != nil {
		return err
	}

	return nil
}

func (b *Booster) Inspect(ctx context.Context, network, addr string, features []protocol.Message) (<-chan interface{}, error) {
	log.Info.Printf("booster: -> inspect: %v", addr)

	conn, err := b.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("booster: unable to connect to (%v): %v", addr, err)
	}

	// compose & send the inspect packet
	h, err := protocol.InspectHeader()
	if err != nil {
		conn.Close()
		return nil, err
	}
	pl, err := protocol.EncodePayloadInspect(features)
	if err != nil {
		conn.Close()
		return nil, err
	}

	p := packet.New()
	_, err = p.AddModule(protocol.ModuleHeader, h, protocol.EncodingProtobuf)
	_, err = p.AddModule(protocol.ModulePayload, pl, protocol.EncodingProtobuf)

	if err != nil {
		conn.Close()
		return nil, err
	}

	if err = conn.Send(p); err != nil {
		conn.Close()
		return nil, err
	}

	// consume every message from the connection in a different goroutine.
	pkts, err := conn.Consume()
	if err != nil {
		conn.Close()
		return nil, err
	}

	stream := make(chan interface{}, 1)
	go func() {
		defer func() {
			close(stream)
		}()

		fail := func(err error) {
			log.Error.Printf("booster: inspect error: %v", err)
			conn.Close()
		}

		for p := range pkts {
			if err = ValidatePacket(p); err != nil {
				fail(err)
				return
			}

			h, err := ExtractHeader(p)
			if err != nil {
				fail(err)
				return
			}

			// take only packets requested
			if !isIn(h.ID, features...) {
				log.Info.Printf("booster: inspect: discarding packet: unexpected header: %v", h)
				continue
			}

			// extract the payload
			praw, err := p.Module(protocol.ModulePayload)
			if err != nil {
				fail(err)
				return
			}

			switch h.ID {
			case protocol.MessageNode:
				pl, err := protocol.DecodePayloadNode(praw.Payload())
				if err != nil {
					fail(err)
					return
				}

				stream <- pl
			case protocol.MessageBandwidth:
				pl, err := protocol.DecodePayloadBandwidth(praw.Payload())
				if err != nil {
					fail(err)
					return
				}

				stream <- pl
			}

		}
	}()

	return stream, nil
}

func isIn(id protocol.Message, ids ...protocol.Message) bool {
	for _, v := range ids {
		if id == v {
			return true
		}
	}
	return false
}
