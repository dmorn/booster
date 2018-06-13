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
		if err := b.Net().Decode(p, m, &h); err != nil {
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

		case protocol.MessageNodeUpdate:
			if bc, ok := conn.(*Conn); ok {
				b.HandleTunnel(ctx, bc, p)
			} else {
				log.Info.Print("booster: discarding packet: this connection cannot tunnel packets")
			}
		case protocol.MessageNotify:
			go b.ServeStatus(ctx, conn)

		case protocol.MessageMonitor:
			go b.ServeMonitor(ctx, conn, p)

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
	if err := b.Net().Decode(p, m, &pl); err != nil {
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
	if err := b.Net().Decode(p, m, &pl); err != nil {
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
	if err := b.Net().Decode(p, m, &pl); err != nil {
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
	if err := b.Net().Decode(p, m, &pl); err != nil {
		fail(err)
		return
	}

	log.Info.Printf("booster: <- disconnect: %v", pl.ID)

	// retrieve the connection we're trying to disconnect from
	c, ok := b.Net().Outgoing[pl.ID]
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
	pl := new(protocol.PayloadProxyUpdate)
	m := protocol.ModulePayload
	if err := b.Net().Decode(p, m, &pl); err != nil {
		fail(err)
		return
	}

	log.Debug.Printf("booster: <- tunnel: %v", pl)

	if err := b.UpdateNode(conn.RemoteNode, *pl, true); err != nil {
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
		Topic: socks5.TopicTunnelEvents,
		Run: func(i interface{}) error {
			// Read every tunnel message, compose a packet with them
			// and send them trough the connection
			pl, ok := i.(protocol.PayloadProxyUpdate)
			if !ok {
				return fmt.Errorf("unable to recognise workload message: %v", pl)
			}

			msg := protocol.MessageNodeUpdate
			p, err := b.Net().EncodeDefault(pl, msg)
			if err != nil {
				return err
			}

			log.Debug.Printf("booster: -> tunnel update: %v", p)

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
		log.Error.Printf("booster: serve status error: %v", err)
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

// ServeMonitor is a blocking function that serves information responding to an monitor package.
// The package should contain a list of supported features that should be delivered.
func (b *Booster) ServeMonitor(ctx context.Context, conn SendCloser, p *packet.Packet) {
	log.Info.Print("booster: <- serving monitor...")

	defer conn.Close()

	fail := func(err error) {
		log.Error.Printf("booster: serve monitor error: %v", err)
	}

	// extract features to serve
	pl := new(protocol.PayloadMonitor)
	m := protocol.ModulePayload
	if err := b.Net().Decode(p, m, &pl); err != nil {
		fail(err)
		return
	}

	var wg sync.WaitGroup
	exec := func(f func(ctx context.Context, conn SendCloser) error) {
		if err := f(ctx, conn); err != nil {
			log.Error.Printf("booster: serve monitor: %v", err)
		}
		wg.Done()
	}

	for _, v := range pl.Features {
		switch v {
		case protocol.MessageNodeUpdate:
			wg.Add(1)
			go exec(b.serveProxy)
		case protocol.MessageNetworkUpdate:
			wg.Add(1)
			go exec(b.serveNet)
		default:
			// do nothing, feature not supported
			log.Info.Printf("booster: serve monitor: feature (%v) not supported", v)
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
			msg := protocol.MessageNetworkUpdate

			p, err := b.Net().Encode(pl, msg, packet.Metadata{
				Encoding:    protocol.EncodingJson,
				Compression: protocol.CompressionNone,
				Encryption:  protocol.EncryptionNone,
			})
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

func (b *Booster) serveProxy(ctx context.Context, conn SendCloser) error {
	sendNode := func(n *node.Node) error {
		tunnels := []*protocol.Tunnel{}
		for _, v := range n.Tunnels() {
			tunnels = append(tunnels, &protocol.Tunnel{
				ID:        v.ID(),
				Target:    v.Target,
				Copies:    v.Copies(),
			})
		}
		node := &protocol.PayloadNode{
			ID:      n.ID(),
			BAddr:   n.BAddr.String(),
			PAddr:   n.PAddr.String(),
			Active:  n.IsActive(),
			Tunnels: tunnels,
		}

		p, err := b.Net().Encode(node, protocol.MessageNodeStatus, packet.Metadata{
			Encoding:    protocol.EncodingJson,
			Compression: protocol.CompressionNone,
			Encryption:  protocol.EncryptionNone,
		})
		if err != nil {
			return err
		}
		if err = conn.Send(p); err != nil {
			return err
		}
		return nil
	}

	// send the initial node state first, then proxy updates
	rootNode, nodes := b.Net().Nodes()
	if err := sendNode(rootNode); err != nil {
		return err
	}
	for _, v := range nodes {
		if err := sendNode(v); err != nil {
			return err
		}
	}

	errc := make(chan error)
	cancel, err := b.Net().Sub(&pubsub.Command{
		Topic: socks5.TopicTunnelEvents,
		Run: func(i interface{}) error {
			ppu, ok := i.(protocol.PayloadProxyUpdate)
			if !ok {
				return fmt.Errorf("unrecognised node message: %v", i)
			}
			p, err := b.Net().Encode(ppu, protocol.MessageNodeUpdate, packet.Metadata{
				Encoding: protocol.EncodingJson,
			})
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
