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

package protocol

import (
	"fmt"
	"strings"
	"time"

	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type PayloadCtrl struct {
	Operation Operation
}

func DecodePayloadCtrl(p []byte) (*PayloadCtrl, error) {
	payload := new(internal.PayloadCtrl)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadCtrl{
		Operation: Operation(payload.Operation),
	}, nil
}

func EncodePayloadCtrl(op Operation) ([]byte, error) {
	p := &internal.PayloadCtrl{
		Operation: int32(op),
	}

	return proto.Marshal(p)
}

type PayloadBandwidth struct {
	Tot       int    `json:"tot"`
	Bandwidth int    `json:"bandwidth"`
	Type      string `json:"type"`
}

func (n *PayloadBandwidth) String() string {
	var b strings.Builder
	b.WriteString("net {\n")

	b.WriteString(fmt.Sprintf("\ttype: %s\n", n.Type))
	b.WriteString(fmt.Sprintf("\tbandwidth: %v, total: %v\n", n.Bandwidth, n.Tot))
	b.WriteString("\n}\n")

	return b.String()
}

func DecodePayloadBandwidth(p []byte) (*PayloadBandwidth, error) {
	payload := new(internal.PayloadBandwidth)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadBandwidth{
		Tot:       int(payload.Tot),
		Bandwidth: int(payload.Bandwidth),
		Type:      payload.Type,
	}, nil
}

func EncodePayloadBandwidth(tot int, bw int, t string) ([]byte, error) {
	p := &internal.PayloadBandwidth{
		Tot:       int64(tot),
		Bandwidth: int64(bw),
		Type:      t,
	}

	return proto.Marshal(p)
}

type PayloadInspect struct {
	Features []Message
}

func DecodePayloadInspect(p []byte) (*PayloadInspect, error) {
	payload := new(internal.PayloadInspect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	features := []Message{}
	for _, v := range payload.Features {
		features = append(features, Message(v))
	}

	return &PayloadInspect{
		Features: features,
	}, nil
}

func EncodePayloadInspect(f []Message) ([]byte, error) {
	features := []int32{}
	for _, v := range f {
		features = append(features, int32(v))
	}

	p := &internal.PayloadInspect{
		Features: features,
	}

	return proto.Marshal(p)
}

type PayloadHello struct {
	BPort string
	PPort string
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

type PayloadConnect struct {
	Target string
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

type PayloadDisconnect struct {
	ID string
}

func DecodePayloadDisconnect(p []byte) (*PayloadDisconnect, error) {
	payload := new(internal.PayloadDisconnect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadDisconnect{
		ID: payload.Id,
	}, nil
}

func EncodePayloadDisconnect(id string) ([]byte, error) {
	p := &internal.PayloadDisconnect{
		Id: id,
	}

	return proto.Marshal(p)
}

type PayloadNode struct {
	ID      string    `json:"id"`
	BAddr   string    `json:"baddr"`
	PAddr   string    `json:"paddr"`
	Active  bool      `json:"active"`
	Tunnels []*Tunnel `json:"tunnels"`
}

func (n *PayloadNode) String() string {
	var b strings.Builder
	b.WriteString("node {\n")

	id := string([]byte(n.ID)[:10])
	b.WriteString(fmt.Sprintf("\tid: %v active: %v\n", id, n.Active))
	b.WriteString(fmt.Sprintf("\tba: %v pa: %v\n", n.BAddr, n.PAddr))
	b.WriteString("\ttunnels:\n")
	b.WriteString("\t[")
	if len(n.Tunnels) == 0 {
		b.WriteString("]")
	} else {
		b.WriteString("\n")
		for _, t := range n.Tunnels {
			b.WriteString(fmt.Sprintf("\t\t%v\n", t))
		}
		b.WriteString("\t]")
	}
	b.WriteString("\n}\n")

	return b.String()
}

type Tunnel struct {
	ID     string `json:"id"`
	Target string `json:"target"`
	Acks   int    `json:"acks"`
	Copies int    `json:"copies"`
}

func (t *Tunnel) String() string {
	id := string([]byte(t.ID)[:10])
	return fmt.Sprintf(
		"{id: %v target: %v acks: %v copies %v}",
		id, t.Target, t.Acks, t.Copies,
	)
}

func DecodePayloadNode(p []byte) (*PayloadNode, error) {
	payload := new(internal.PayloadNode)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	ts := []*Tunnel{}
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
	ts := []*internal.PayloadNode_Tunnel{}
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

type PayloadHeartbeat struct {
	ID   string
	Hops int
	TTL  time.Time
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

type PayloadTunnelEvent struct {
	Target string
	Event  int
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

func EncodePayloadTunnelEvent(target string, event int) ([]byte, error) {
	p := &internal.PayloadTunnelEvent{
		Target: target,
		Event:  int32(event),
	}

	return proto.Marshal(p)
}
