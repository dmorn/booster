package booster

import (
	"context"
	"fmt"

	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

func (b *Booster) SendStatus(ctx context.Context, conn SendCloser) error {
	// register for proxy updates
	c, err := b.Proxy.Notify()
	if err != nil {
		return err
	}

	defer func() {
		b.Proxy.StopNotifying(c)
	}()

	// TODO(daniel): read every tunnel message, compose a packet with them
	// and send them trough the connection
	return fmt.Errorf("status not yet implemented")
}

func (b *Booster) SendHello(ctx context.Context, conn SendCloser) error {
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

	// TODO(daniel): should we check if the node actually removed this node?
	return nil
}
