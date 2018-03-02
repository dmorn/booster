package booster

import (
	"context"

	"github.com/danielmorandini/booster/network"
	"github.com/danielmorandini/booster/protocol"
	"github.com/danielmorandini/booster/network/packet"
)

func (b *Booster) SendHello(ctx context.Context, conn *network.Conn) error {
	// compose the packet
	h, err := protocol.HelloHeader()
	if err != nil {
		return err
	}

	pl, err := protocol.HelloPayload()
	if err != nil {
		return err
	}

	p := packet.New()
	if _, err = p.AddModule(protocol.ModuleHeader, h, protocol.EncodingProtobuf); err != nil {
		return err
	}

	if _, err = p.AddModule(protocol.ModulePayload, pl, protocol.EncodingProtobuf); err != nil {
		return err
	}

	return conn.Send(p)
}
