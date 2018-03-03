package booster

import (
	"context"
	"fmt"
	"net"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
)

func RecvHello(ctx context.Context, conn *network.Conn) (*Conn, error) {
	fail := func(err error) (*Conn, error) {
		conn.Close()
		return nil, err
	}

	// Read the hello packet
	p, err := conn.Recv()
	if err != nil {
		return fail(err)
	}

	// Find header
	hraw, err := p.Module(protocol.ModuleHeader)
	if err != nil {
		return fail(err)
	}

	h, err := protocol.DecodeHeader(hraw.Payload())
	if err != nil {
		return fail(err)
	}

	// check that it is a hello message
	if h.ID != protocol.MessageHello {
		return fail(fmt.Errorf("booster: expected HelloMessage (%v), found: %v", protocol.MessageHello, h.ID))
	}

	// check what the header says about the package before trying to take
	// the payload
	if err = ValidatePacket(p); err != nil {
		return fail(fmt.Errorf("booster: hello: %v", err))
	}

	// take the payload module
	praw, err := p.Module(protocol.ModulePayload)
	if err != nil {
		return fail(err)
	}

	pl, err := protocol.DecodePayloadHello(praw.Payload())
	if err != nil {
		return fail(err)
	}

	// extract node information from the message
	pp := pl.PPort
	bp := pl.BPort
	host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())

	// create new node with the information collected
	node, err := node.New(host, pp, bp, false)
	if err != nil {
		return fail(err)
	}

	return &Conn{
		ID:         node.ID(),
		Conn:       conn,
		RemoteNode: node,
	}, nil
}

func (b *Booster) HandleConnect(ctx context.Context, p *packet.Packet) (*packet.Packet, error) {
	if err := ValidatePacket(p); err != nil {
		return nil, fmt.Errorf("booster: connect: %v", err)
	}

	// extract information
	praw, err := p.Module(protocol.ModulePayload)
	if err != nil {
		return nil, err
	}

	pl, err := protocol.DecodePayloadConnect(praw.Payload())
	if err != nil {
		return nil, err
	}

	conn, err := b.Wire(ctx, "tcp", pl.Target)
	if err != nil {
		return nil, err
	}

	// TODO(daniel): send the connected node as response
	h, err := protocol.NodeHeader()
	if err != nil {
		return nil, err
	}

	n := conn.RemoteNode
	tunnels := make([]*protocol.Tunnel, len(n.Tunnels()))
	for _, t := range n.Tunnels() {
		tunnel := &protocol.Tunnel{
			ID: t.ID(),
			Target: t.Target,
			Acks: t.Acks(),
			Copies: t.Copies(),
		}

		tunnels = append(tunnels, tunnel)
	}
	param := &protocol.PayloadNode{
		ID: n.ID(),
		BAddr: n.BAddr.String(),
		PAddr: n.PAddr.String(),
		Active: n.IsActive(),
		Tunnels: tunnels,
	}

	npl, err := protocol.EncodePayloadNode(param)
	if err != nil {
		return nil, err
	}

	p = packet.New()
	enc := protocol.EncodingProtobuf
	if _, err = p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		return nil, err
	}
	if _, err = p.AddModule(protocol.ModulePayload, npl, enc); err != nil {
		return nil, err
	}

	return p, nil
}
