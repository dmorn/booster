package protocol

import (
	"time"

	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type PayloadHello struct {
	BPort string
	PPort string
}

type PayloadConnect struct {
	Target string
}

type Tunnel struct {
	ID     string
	Target string
	Acks   int
	Copies int
}

type PayloadNode struct {
	ID      string
	BAddr   string
	PAddr   string
	Active  bool
	Tunnels []*Tunnel
}

type PayloadHeartbeat struct {
	ID   string
	Hops int
	TTL  time.Time
}

type PayloadTunnelEvent struct {
	Target string
	Event  int
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

	return &PayloadConnect{
		Target: payload.Target,
	}, nil
}

func EncodePayloadConnect(target string) ([]byte, error) {
	p := &internal.PayloadConnect{
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
			ID:     t.Id,
			Target: t.Target,
			Acks:   int(t.Acks),
			Copies: int(t.Copies),
		}

		ts = append(ts, tunnel)
	}

	return &PayloadNode{
		ID:      payload.Id,
		BAddr:   payload.Baddr,
		PAddr:   payload.Paddr,
		Active:  payload.Active,
		Tunnels: ts,
	}, nil
}

func EncodePayloadNode(node *PayloadNode) ([]byte, error) {
	ts := make([]*internal.PayloadNode_Tunnel, len(node.Tunnels))
	for _, t := range node.Tunnels {
		tunnel := &internal.PayloadNode_Tunnel{
			Id:     t.ID,
			Target: t.Target,
			Acks:   int32(t.Acks),
			Copies: int32(t.Copies),
		}

		ts = append(ts, tunnel)
	}

	p := &internal.PayloadNode{
		Id:      node.ID,
		Baddr:   node.BAddr,
		Paddr:   node.PAddr,
		Active:  node.Active,
		Tunnels: ts,
	}

	return proto.Marshal(p)
}

func DecodePayloadHeartbeat(p []byte) (*PayloadHeartbeat, error) {
	payload := new(internal.PayloadHeartbeat)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	t, err := ptypes.Timestamp(payload.Ttl)
	if err != nil {
		return nil, err
	}

	return &PayloadHeartbeat{
		ID:   payload.Id,
		Hops: int(payload.Hops),
		TTL:  t,
	}, nil
}

func EncodePayloadHeartbeat(h *PayloadHeartbeat) ([]byte, error) {
	t, err := ptypes.TimestampProto(h.TTL)
	if err != nil {
		return nil, err
	}

	p := &internal.PayloadHeartbeat{
		Id:   h.ID,
		Ttl:  t,
		Hops: int32(h.Hops),
	}

	return proto.Marshal(p)
}

func DecodePayloadTunnelEvent(p []byte) (*PayloadTunnelEvent, error) {
	payload := new(internal.PayloadTunnelEvent)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadTunnelEvent{
		Target: payload.Target,
		Event:  int(payload.Event),
	}, nil
}

func EncodePaylaodTunnelEvent(target string, event int) ([]byte, error) {
	p := &internal.PayloadTunnelEvent{
		Target: target,
		Event:  int32(event),
	}

	return proto.Marshal(p)
}
