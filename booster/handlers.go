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
	"sync"
	"time"

	"github.com/danielmorandini/booster/log"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/node"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/pubsub"
	"github.com/danielmorandini/booster/socks5"
)

// Handle takes care of consuming the connection, handling each incoming packet.
// Blocks until conn gets closed.
func (b *Booster) Handle(ctx context.Context, conn SendConsumeCloser) {
	// consume the connection until pkts is closed
	pkts, err := conn.Consume()
	if err != nil {
		conn.Close()
		log.Error.Printf("booster: unable to consume conn: %v", err)
		return
	}

	handler := func(p *packet.Packet) error {
		m := protocol.ModuleHeader
		h := new(protocol.Header)
		f := protocol.HeaderDecoder
		if err := b.Net().Decode(p, m, &h, f); err != nil {
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
				log.Info.Print("booster: discarding packet: this connection cannot tunnel packets")
			}
		case protocol.MessageNotify:
			go b.ServeStatus(ctx, conn)

		case protocol.MessageInspect:
			go b.ServeInspect(ctx, conn, p)

		case protocol.MessageCtrl:
			b.HandleCtrl(ctx, conn, p)

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

// HandleCtrl handels control (ctrl) packets.
func (b *Booster) HandleCtrl(ctx context.Context, conn SendCloser, p *packet.Packet) {
	fail := func(err error) {
		log.Error.Printf("booster: ctrl error: %v", err)
		conn.Close()
	}

	// extract information
	pl := new(protocol.PayloadCtrl)
	m := protocol.ModulePayload
	f := protocol.PayloadDecoders[protocol.MessageCtrl]
	if err := b.Net().Decode(p, m, &pl, f); err != nil {
		fail(err)
		return
	}

	// check the control operation that we have to perform
	switch pl.Operation {
	case protocol.CtrlStop:
		if err := b.Close(); err != nil {
			fail(err)
			return
		}
	case protocol.CtrlRestart:
		if err := b.Restart(); err != nil {
			fail(err)
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

	// extract information
	pl := new(protocol.PayloadHeartbeat)
	m := protocol.ModulePayload
	f := protocol.PayloadDecoders[protocol.MessageHeartbeat]
	if err := b.Net().Decode(p, m, &pl, f); err != nil {
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
		c.HeartbeatTimer.Reset(b.Net().HeartbeatTTL * 2)
	}

	// compose a new heartbeat message
	p, err := b.Net().composeHeartbeat(pl)
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

	// extract information
	pl := new(protocol.PayloadConnect)
	m := protocol.ModulePayload
	f := protocol.PayloadDecoders[protocol.MessageConnect]
	if err := b.Net().Decode(p, m, &pl, f); err != nil {
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
	p, err = b.Net().composeNode(tc.RemoteNode)
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

	// extract information
	pl := new(protocol.PayloadDisconnect)
	m := protocol.ModulePayload
	f := protocol.PayloadDecoders[protocol.MessageDisconnect]
	if err := b.Net().Decode(p, m, &pl, f); err != nil {
		fail(err)
		return
	}

	log.Info.Printf("booster: <- disconnect: %v", pl.ID)

	// retrieve the connection we're trying to disconnect from
	c, ok := b.Net().Conns[pl.ID]
	if !ok {
		fail(fmt.Errorf("unexpected identifier [%v]", pl.ID))
		return
	}

	// do not trace this node as it was manually disconnected
	c.RemoteNode.ToBeTraced = false

	// perform the actual disconnection
	c.Close()

	// TODO(daniel): is this response appropriate?
	// send back a node packet with the info about the
	// disconncted node
	p, err := b.Net().composeNode(c.RemoteNode)
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

	// extract information
	pl := new(protocol.PayloadTunnelEvent)
	m := protocol.ModulePayload
	f := protocol.PayloadDecoders[protocol.MessageTunnel]
	if err := b.Net().Decode(p, m, &pl, f); err != nil {
		fail(err)
		return
	}

	log.Debug.Printf("booster: <- tunnel: %v", pl)

	tm := &socks5.TunnelMessage{
		Target: pl.Target,
		Event:  socks5.Event(pl.Event),
	}
	if err := b.UpdateNode(conn.RemoteNode, tm, true); err != nil {
		fail(err)
		return
	}
}

// ServeStatus listens on the local proxy for tunnel events, sending then them though the
// connection. In case of error closes the connection.
func (b *Booster) ServeStatus(ctx context.Context, conn SendCloser) {
	log.Info.Print("booster: <- status...")

	defer conn.Close()

	errc := make(chan error)
	cancel, err := b.Proxy.Sub(&pubsub.Command{
		Topic: socks5.TopicTunnelUpdates,
		Run: func(i interface{}) error {
			// Read every tunnel message, compose a packet with them
			// and send them trough the connection
			tm, ok := i.(socks5.TunnelMessage)
			if !ok {
				return fmt.Errorf("unable to recognise workload message: %v", tm)
			}

			pl := protocol.PayloadTunnelEvent{
				Target: tm.Target,
				Event:  int(tm.Event),
			}
			msg := protocol.MessageTunnel

			p, err := b.Net().Encode(pl, msg)
			if err != nil {
				return err
			}
			log.Debug.Printf("booster: -> tunnel update: %v", tm)

			if err = conn.Send(p); err != nil {
				return err
			}
			return nil
		},
		PostRun: func(err error) {
			if err != nil {
				errc <- err
			}
		},
	})
	if err != nil {
		errc <- err
	}

	fail := func(err error) {
		log.Error.Printf("booster: serve status error: %v", ctx.Err())
	}

	select {
	case err := <-errc:
		fail(err)
	case <-ctx.Done():
		fail(ctx.Err())
		<-errc
		cancel()
	}
}

// ServeInspect is a blocking function that serves information responding to an inspect package.
// The package should contain a list of supported features that should be delivered.
func (b *Booster) ServeInspect(ctx context.Context, conn SendCloser, p *packet.Packet) {
	log.Info.Print("booster: <- serving inspect...")

	defer conn.Close()

	fail := func(err error) {
		log.Error.Printf("booster: serve inspect error: %v", err)
	}

	// extract features to serve
	pl := new(protocol.PayloadInspect)
	m := protocol.ModulePayload
	f := protocol.PayloadDecoders[protocol.MessageInspect]
	if err := b.Net().Decode(p, m, &pl, f); err != nil {
		fail(err)
		return
	}

	var wg sync.WaitGroup
	exec := func(f func(ctx context.Context, conn SendCloser) error) {
		if err := f(ctx, conn); err != nil {
			log.Error.Printf("booster: serve inspect: %v", err)
		}
		wg.Done()
	}

	for _, v := range pl.Features {
		switch v {
		case protocol.MessageNode:
			wg.Add(1)
			go exec(b.serveNode)
		case protocol.MessageBandwidth:
			wg.Add(1)
			go exec(b.serveNet)
		default:
			// do nothing, feature not supported
			log.Info.Printf("booster: serve inspect: feature (%v) not supported", v)
		}
	}

	wg.Wait()
}

func (b *Booster) serveNet(ctx context.Context, conn SendCloser) error {
	errc := make(chan error)
	cancel, err := b.Proxy.Sub(&pubsub.Command{
		Topic: socks5.TopicNet,
		Run: func(i interface{}) error {
			bm, ok := i.(*socks5.BandwidthMessage)
			if !ok {
				return fmt.Errorf("unrecognised bandwidth message: %v", i)
			}
			t := "download"
			if !bm.Download {
				t = "upload"
			}

			pl := protocol.PayloadBandwidth{
				Tot:       bm.Tot,
				Bandwidth: bm.Bandwidth,
				Type:      t,
			}
			msg := protocol.MessageBandwidth

			p, err := b.Net().Encode(pl, msg)
			if err != nil {
				return err
			}

			if err = conn.Send(p); err != nil {
				return err
			}
			return nil
		},
		PostRun: func(err error) {
			if err != nil {
				errc <- err
			}
		},
	})
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		cancel()
		<-errc
		return ctx.Err()
	case err := <-errc:
		return err
	}
}

func (b *Booster) serveNode(ctx context.Context, conn SendCloser) error {
	errc := make(chan error)
	cancel, err := b.Net().Sub(&pubsub.Command{
		Topic: TopicNode,
		Run: func(i interface{}) error {
			n, ok := i.(*node.Node)
			if !ok {
				return fmt.Errorf("unrecognised node message: %v", i)
			}

			p, err := b.Net().composeNode(n)
			if err != nil {
				return err
			}

			if err = conn.Send(p); err != nil {
				return err
			}
			return nil
		},
		PostRun: func(err error) {
			if err != nil {
				errc <- err
			}
		},
	})
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		cancel()
		<-errc
		return ctx.Err()
	case err := <-errc:
		return err
	}
}
