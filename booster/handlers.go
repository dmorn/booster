package booster

import (
	"context"
	"fmt"
	"time"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/socks5"
)

func (b *Booster) Handle(ctx context.Context, conn SendConsumeCloser) {
	// consume the connection until pkts is closed
	pkts, err := conn.Consume()
	if err != nil {
		conn.Close()
		log.Error.Printf("booster: unable to consume conn: %v", err)
		return
	}

	handler := func(p *packet.Packet) error {
		h, err := ExtractHeader(p)
		if err != nil {
			return err
		}

		// find the message type and handle accordingly
		switch h.ID {
		case protocol.MessageConnect:
			b.HandleConnect(ctx, conn, p)

		case protocol.MessageDisconnect:
			b.HandleDisconnect(ctx, conn, p)

		case protocol.MessageHeartbeat:
			b.HandleHeartbeat(ctx, conn, p)

		case protocol.MessageTunnel:
			if bc, ok := conn.(*Conn); ok {
				b.HandleTunnel(ctx, bc, p)
			} else {
				log.Info.Printf("booster: discarding packet: this connection cannot tunnel packets")
			}
		case protocol.MessageNotify:
			go b.ServeStatus(ctx, conn)

		case protocol.MessageInspect:
			go b.ServeInspect(ctx, conn)

		default:
			return fmt.Errorf("booster: discarding packet: unexpected message id: %v", h.ID)
		}

		return nil
	}

	for p := range pkts {
		if err := handler(p); err != nil {
			log.Error.Println(err)
			conn.Close()
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
	fail := func(err error) {
		log.Error.Printf("booster: heartbeat error: %v", err)
		conn.Close()
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
	pl, err := protocol.DecodePayloadHeartbeat(praw.Payload())
	if err != nil {
		fail(err)
		return
	}

	// check that we received the heartbeat message in time
	if pl.TTL.Before(time.Now()) {
		fail(fmt.Errorf("heartbeat message TTL expired: TTL %v, Now %v", pl.TTL, time.Now()))
		return
	}

	// stop the timer or it will close the connection
	if c, ok := conn.(*Conn); ok {
		c.HeartbeatTimer.Stop()
	}

	// wait until ttl finishes & reset the timer
	<-time.After(pl.TTL.Sub(time.Now()))
	if c, ok := conn.(*Conn); ok {
		c.HeartbeatTimer.Reset(b.HeartbeatTTL * 2)
	}

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
// Should run in its own gorountine. Closes the connection used to perform the
// request when done.
func (b *Booster) HandleConnect(ctx context.Context, conn SendCloser, p *packet.Packet) {
	// TODO: add some more information to the errors printed.
	defer func() {
		conn.Close()
	}()

	fail := func(err error) {
		log.Error.Printf("booster: connect error: %v", err)
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

	log.Info.Printf("booster: <- connect: %v", pl.Target)

	// send back a node packet with the info about the
	// newly connected node
	p, err = composeNode(tc.RemoteNode)
	if err != nil {
		fail(err)
		return
	}

	if err = conn.Send(p); err != nil {
		fail(err)
		return
	}
}

// HandleDisconnect takes a disconnect packet, unwraps its information and removes the
// target node in this node's network, closing the connection between the two.
//
// Disconnect also closes the connection used to perform this operation upon error or
// when done.
func (b *Booster) HandleDisconnect(ctx context.Context, conn SendCloser, p *packet.Packet) {
	// TODO: add some more information to the errors printed.
	defer func() {
		conn.Close()
	}()

	fail := func(err error) {
		log.Error.Printf("booster: disconnect error: %v", err)
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

	pl, err := protocol.DecodePayloadDisconnect(praw.Payload())
	if err != nil {
		fail(err)
		return
	}

	log.Info.Printf("booster: <- disconnect: %v", pl.ID)

	// retrieve the connection we're trying to disconnect from
	c, ok := Nets.Get(b.ID).Conns[pl.ID]
	if !ok {
		fail(fmt.Errorf("unexpected identifier [%v]", pl.ID))
		return
	}

	// do not trace this node as it was manually disconnected
	c.RemoteNode.ToBeTraced = false

	// perform the actual disconnection
	if err = c.Close(); err != nil {
		fail(err)
		return
	}

	// TODO(daniel): is this response appropriate?
	// send back a node packet with the info about the
	// disconncted node
	p, err = composeNode(c.RemoteNode)
	if err != nil {
		fail(err)
		return
	}

	if err = conn.Send(p); err != nil {
		fail(err)
		return
	}
}

func (b *Booster) HandleTunnel(ctx context.Context, conn *Conn, p *packet.Packet) {
	// TODO: add some more information to the errors printed.
	fail := func(err error) {
		log.Error.Printf("booster: tunnel error: %v", err)
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

	pl, err := protocol.DecodePayloadTunnelEvent(praw.Payload())
	if err != nil {
		fail(err)
		return
	}

	log.Info.Printf("booster: <- tunnel: %v", pl)

	tm := &socks5.TunnelMessage{
		Target: pl.Target,
		Event:  socks5.Event(pl.Event),
	}
	if err = b.UpdateNode(conn.RemoteNode, tm, true); err != nil {
		fail(err)
		return
	}
}

// ServeStatus listens on the local proxy for tunnel events, sending then them though the
// connection. In case of error closes the connection.
func (b *Booster) ServeStatus(ctx context.Context, conn SendCloser) {
	log.Info.Print("booster: <- status...")

	fail := func(err error) {
		log.Error.Printf("booster: serve status error: %v", err)
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

		log.Info.Printf("booster: -> tunnel update: %v", tm)

		if err = conn.Send(p); err != nil {
			fail(err)
			return
		}
	}
}

func (b *Booster) ServeInspect(ctx context.Context, conn SendCloser) {
	log.Info.Print("booster: <- serving inspect...")

	fail := func(err error) {
		log.Error.Printf("booster: serve inspect error: %v", err)
		conn.Close()
	}

	// register for node updates
	net := Nets.Get(b.ID)
	c, err := net.Notify()
	if err != nil {
		fail(err)
		return
	}

	defer func() {
		net.StopNotifying(c)
	}()

	// Read every node udpate message, compose a packet with it
	// and send them trough the connection
	for i := range c {
		n, ok := i.(*node.Node)
		if !ok {
			fail(fmt.Errorf("unrecognised node message: %v", i))
			return
		}

		p, err := composeNode(n)
		if err != nil {
			fail(err)
			return
		}

		if err = conn.Send(p); err != nil {
			fail(err)
			return
		}
	}
}
