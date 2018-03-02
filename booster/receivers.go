package booster

import (
	"context"
	"fmt"
	"net"

	"github.com/danielmorandini/booster/network"
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
	if !h.HasModule(protocol.ModulePayload) {
		return fail(fmt.Errorf("booster: hello message does not contain any payload"))
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
