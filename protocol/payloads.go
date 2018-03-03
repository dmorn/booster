package protocol

import (
	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
)

type PayloadHello struct {
	BPort string
	PPort string
}

type PayloadConnect struct {
	Target string
}

type Tunnel struct {
	ID string
	Target string
	Acks int
	Copies int
}

type PayloadNode struct {
	ID string
	BAddr string
	PAddr string
	Active bool
	Tunnels []*Tunnel
}

func DecodePayloadHello(p []byte) (*PayloadHello, error) {
	payload := new(internal.PayloadHello)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadHello{
		BPort: payload.Bport,
		PPort: payload.Pport,
	}, nil
}

func EncodePayloadHello(bport, pport string) ([]byte, error) {
	p := &internal.PayloadHello{
		Pport: pport,
		Bport: bport,
	}

	return proto.Marshal(p)
}

func DecodePayloadConnect(p []byte) (*PayloadConnect, error) {
	payload := new(internal.PayloadConnect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadConnect {
		Target: payload.Target,
	}, nil
}

func EncodePayloadConnect(target string) ([]byte, error) {
	p := &internal.PayloadConnect {
		Target: target,
	}

	return proto.Marshal(p)
}

func DecodePayloadNode(p []byte) (*PayloadNode, error) {
	payload := new(internal.PayloadNode)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	ts := make([]*Tunnel, len(payload.Tunnels))
	for _, t := range payload.Tunnels {
		tunnel := &Tunnel{
			ID: t.Id,
			Target: t.Target,
			Acks: int(t.Acks),
			Copies: int(t.Copies),
		}

		ts = append(ts, tunnel)
	}

	return &PayloadNode{
		ID: payload.Id,
		BAddr: payload.Baddr,
		PAddr: payload.Paddr,
		Active: payload.Active,
		Tunnels: ts,
	}, nil
}

func EncodePayloadNode(node *PayloadNode) ([]byte, error) {
	ts := make([]*internal.PayloadNode_Tunnel, len(node.Tunnels))
	for _, t := range node.Tunnels {
		tunnel := &internal.PayloadNode_Tunnel{
			Id: t.ID,
			Target: t.Target,
			Acks: int32(t.Acks),
			Copies: int32(t.Copies),
		}

		ts = append(ts, tunnel)
	}

	p := &internal.PayloadNode{
		Id: node.ID,
		Baddr: node.BAddr,
		Paddr: node.PAddr,
		Active: node.Active,
		Tunnels: ts,
	}

	return proto.Marshal(p)
}
