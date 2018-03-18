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

func (b *Booster) Inspect(ctx context.Context, network, addr string) (<-chan *protocol.PayloadNode, error) {
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
	p := packet.New()
	if _, err = p.AddModule(protocol.ModuleHeader, h, protocol.EncodingProtobuf); err != nil {
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

	stream := make(chan *protocol.PayloadNode, 1)
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
			// take only node packets
			if h.ID != protocol.MessageNode {
				log.Info.Printf("booster: inspect: discarding packet: unexpected header: %v", h)
				continue
			}

			// extract node from payload
			praw, err := p.Module(protocol.ModulePayload)
			if err != nil {
				fail(err)
				return
			}
			pl, err := protocol.DecodePayloadNode(praw.Payload())
			if err != nil {
				fail(err)
				return
			}

			stream <- pl
		}
	}()

	return stream, nil
}
