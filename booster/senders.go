package booster

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/socks5"
)

func (b *Booster) ServeStatus(ctx context.Context, conn SendCloser) {
	b.Println("booster: -> serving status...")

	fail := func(err error) {
		b.Printf("booster: serve status error: %v", err)
		conn.Close()
	}

	// register for proxy updates
	c, err := b.Proxy.Notify()
	if err != nil {
		fail(err)
		return
	}

	defer func() {
		b.Proxy.StopNotifying(c)
	}()

	// Read every tunnel message, compose a packet with them
	// and send them trough the connection
	h, err := protocol.TunnelEventHeader()
	if err != nil {
		fail(err)
		return
	}

	for i := range c {
		tm, ok := i.(socks5.TunnelMessage)
		if !ok {
			fail(fmt.Errorf("unable to recognise workload message: %v", tm))
			return
		}

		pl, err := protocol.EncodePayloadTunnelEvent(tm.Target, int(tm.Event))
		if err != nil {
			fail(err)
			return
		}

		p := packet.New()
		enc := protocol.EncodingProtobuf
		_, err = p.AddModule(protocol.ModuleHeader, h, enc)
		_, err = p.AddModule(protocol.ModulePayload, pl, enc)
		if err != nil {
			fail(err)
			return
		}

		b.Printf("booster: -> tunnel update: %v", tm)
		if err = conn.Send(p); err != nil {
			fail(err)
			return
		}
	}
}

func (b *Booster) SendHello(ctx context.Context, conn SendCloser) error {
	b.Println("booster: -> hello")

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

func (b *Booster) Connect(ctx context.Context, network, laddr, raddr string) (string, error) {
	b.Println("booster: -> connect")

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

func (b *Booster) Disconnect(ctx context.Context, network, addr, id string) error {
	b.Println("booster: -> disconnect")

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
