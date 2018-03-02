package booster

import (
	"context"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

func (b *Booster) SendHello(ctx context.Context, conn *network.Conn) error {
	// create the modules
	h, err := protocol.HelloHeader()
	if err != nil {
		return err
	}

	n := b.Network.LocalNode
	pp := n.PPort()
	bp := n.BPort()

	pl, err := protocol.HelloPayload(bp, pp)
	if err != nil {
		return err
	}

	// compose the packet
	p := packet.New()
	if _, err = p.AddModule(protocol.ModuleHeader, h, protocol.EncodingProtobuf); err != nil {
		return err
	}
	if _, err = p.AddModule(protocol.ModulePayload, pl, protocol.EncodingProtobuf); err != nil {
		return err
	}

	// send
	return conn.Send(p)
}
