package booster

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
)

func (b *Booster) Handle(ctx context.Context, conn SendConsumeCloser) {
	b.Println("booster: -> handle")
	defer func() {
		b.Println("booster: <- handle")
	}()

	// consume the connection until pkts is closed
	pkts, err := conn.Consume()
	if err != nil {
		b.Printf("booster: unable to consume conn: %v", err)
		return
	}

	handler := func(p *packet.Packet) error {
		h, err := ExtractHeader(p)
		if err != nil {
			b.Println(err)
			return nil
		}

		// find the message type and handle accordingly
		switch h.ID {
		case protocol.MessageConnect:
			b.HandleConnect(ctx, conn, p)

		case protocol.MessageHeartbeat:
			b.HandleHeartbeat(ctx, conn, p)

		default:
			b.Printf("booster: discarding packet: unexpected message id: %v", h.ID)
		}

		return nil
	}

	for p := range pkts {
		if err := handler(p); err != nil {
			b.Println(err)
			return
		}
	}
}

// HandleHeartbeat validates the packet and checks the validity and expiration
// of its payload. If the TTL is not yet expired, it waits for it to finish before
// composing a new heartbeat message, in order to avoid a flood.
//
// Closes the connection in case of any kind of failure.
func (b *Booster) HandleHeartbeat(ctx context.Context, conn SendCloser, p *packet.Packet) {
	b.Println("booster: -> heartbeat")
	defer func() {
		b.Print("booster: <- heartbeat")
	}()

	fail := func(err error) {
		b.Printf("booster: heartbeat error: %v", err)
		conn.Close()
	}

	if err := ValidatePacket(p); err != nil {
		fail(fmt.Errorf("booster: connect: %v", err))
		return
	}

	// extract information
	praw, err := p.Module(protocol.ModulePayload)
	if err != nil {
		fail(err)
		return
	}
	pl, err := protocol.DecodePayloadHeartbeat(praw.Payload())
	if err != nil {
		fail(err)
		return
	}

	// check that we received the heartbeat message in time
	if pl.TTL.Before(time.Now()) {
		fail(fmt.Errorf("heartbeat message TTL expired: %v", pl.TTL))
		return
	}

	// wait until ttl finishes
	<-time.After(pl.TTL.Sub(time.Now()))

	// compose a new heartbeat message
	p, err = b.composeHeartbeat(pl)
	if err != nil {
		fail(err)
		return
	}

	// send it
	if err = conn.Send(p); err != nil {
		fail(err)
		return
	}
}

// HandleConnect handles a connect packet. It validates the packet and retrives the
// target node from its payload. It then wires with the target node, handling the new
// connection in a different goroutine.
//
// If the connection with the remote node is successfull, sends a node packet with
// the information regarding the added node back. The node identifier contained in
// the packet is used as connection identifier in the network.
//
// Should run in its own gorounting. Closes the connection used to perform the
// request when done.
func (b *Booster) HandleConnect(ctx context.Context, conn SendCloser, p *packet.Packet) {
	// TODO: add some more information to the errors printed.
	b.Println("booster: -> connect")
	defer func() {
		conn.Close()
		b.Println("booster: <- connect")
	}()

	fail := func(err error) {
		b.Printf("booster: connect error: %v", err)
	}

	if err := ValidatePacket(p); err != nil {
		fail(err)
		return
	}

	// extract information
	praw, err := p.Module(protocol.ModulePayload)
	if err != nil {
		fail(err)
		return
	}

	pl, err := protocol.DecodePayloadConnect(praw.Payload())
	if err != nil {
		fail(err)
		return
	}

	tc, err := b.Wire(ctx, "tcp", pl.Target) // target connection
	if err != nil {
		fail(err)
		return
	}

	// send back a node packet with the info about the
	// newly connected node
	h, err := protocol.NodeHeader()
	if err != nil {
		fail(err)
		return
	}

	n := tc.RemoteNode
	tunnels := make([]*protocol.Tunnel, len(n.Tunnels()))
	for _, t := range n.Tunnels() {
		tunnel := &protocol.Tunnel{
			ID:     t.ID(),
			Target: t.Target,
			Acks:   t.Acks(),
			Copies: t.Copies(),
		}

		tunnels = append(tunnels, tunnel)
	}
	param := &protocol.PayloadNode{
		ID:      n.ID(),
		BAddr:   n.BAddr.String(),
		PAddr:   n.PAddr.String(),
		Active:  n.IsActive(),
		Tunnels: tunnels,
	}

	npl, err := protocol.EncodePayloadNode(param)
	if err != nil {
		fail(err)
		return
	}

	p = packet.New()
	enc := protocol.EncodingProtobuf
	if _, err = p.AddModule(protocol.ModuleHeader, h, enc); err != nil {
		fail(err)
		return
	}
	if _, err = p.AddModule(protocol.ModulePayload, npl, enc); err != nil {
		fail(err)
		return
	}

	if err = conn.Send(p); err != nil {
		fail(err)
		return
	}
}

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
