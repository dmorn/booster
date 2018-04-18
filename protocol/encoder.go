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
	"reflect"

	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

// EncoderFunc defines how an encoder should behave.
type EncoderFunc func(interface{}) ([]byte, error)

// Implemented default encoders
var encoders = map[Message]EncoderFunc{
	MessageHello:      encodeHello,
	MessageCtrl:       encodeCtrl,
	MessageBandwidth:  encodeBandwidth,
	MessageInspect:    encodeInspect,
	MessageConnect:    encodeConnect,
	MessageDisconnect: encodeDisconnect,
	MessageNode:       encodeNode,
	MessageHeartbeat:  encodeHeartbeat,
	MessageTunnel:     encodeTunnelEvent,
}

// Encoder wraps a list of implemented encoder functions.
type Encoder struct {
	Encoders map[Message]EncoderFunc
}

// NewEncoder retusn a new instance of Encoder, filled with a default list
// of encoder functions ready to be used.
func NewEncoder() *Encoder {
	return &Encoder{
		Encoders: encoders,
	}
}

// Encode takes msg and tries to retrieve an encoder function for it.
// It then encodes v using that function.
//
// v has to be a value, not a pointer (in fact we don't want v to be
// modified by this function in any way)
func (e *Encoder) Encode(v interface{}, msg Message) ([]byte, error) {
	// retrieve the right encoder function
	f, ok := e.Encoders[msg]
	if !ok {
		return nil, fmt.Errorf("protocol: encode error: could find any encode function for message (%v)", msg)
	}

	return f(v)
}

func conversionFail(v interface{}) error {
	return fmt.Errorf("protocol: encode error: unable to make type assertion: %v is of unexpected type %v", v, reflect.TypeOf(v))
}

func encodeHello(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadHello)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadHello{
		Pport: d.PPort,
		Bport: d.BPort,
	}

	return proto.Marshal(p)
}

func encodeCtrl(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadCtrl)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadCtrl{
		Operation: int32(d.Operation),
	}

	return proto.Marshal(p)
}

func encodeBandwidth(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadBandwidth)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadBandwidth{
		Tot:       int64(d.Tot),
		Bandwidth: int64(d.Bandwidth),
		Type:      d.Type,
	}

	return proto.Marshal(p)
}

func encodeInspect(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadInspect)
	if !ok {
		return nil, conversionFail(v)
	}

	features := []int32{}
	for _, v := range d.Features {
		features = append(features, int32(v))
	}

	p := &internal.PayloadInspect{
		Features: features,
	}

	return proto.Marshal(p)
}

func encodeConnect(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadConnect)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadConnect{
		Target: d.Target,
	}

	return proto.Marshal(p)
}

func encodeDisconnect(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadDisconnect)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadDisconnect{
		Id: d.ID,
	}

	return proto.Marshal(p)
}

func encodeNode(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadNode)
	if !ok {
		return nil, conversionFail(v)
	}

	ts := []*internal.PayloadNode_Tunnel{}
	for _, t := range d.Tunnels {
		tunnel := &internal.PayloadNode_Tunnel{
			Id:     t.ID,
			Target: t.Target,
			Acks:   int32(t.Acks),
			Copies: int32(t.Copies),
		}

		ts = append(ts, tunnel)
	}

	p := &internal.PayloadNode{
		Id:      d.ID,
		Baddr:   d.BAddr,
		Paddr:   d.PAddr,
		Active:  d.Active,
		Tunnels: ts,
	}

	return proto.Marshal(p)
}

func encodeHeartbeat(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadHeartbeat)
	if !ok {
		return nil, conversionFail(v)
	}

	t, err := ptypes.TimestampProto(d.TTL)
	if err != nil {
		return nil, err
	}

	p := &internal.PayloadHeartbeat{
		Id:   d.ID,
		Ttl:  t,
		Hops: int32(d.Hops),
	}

	return proto.Marshal(p)
}

func encodeTunnelEvent(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadTunnelEvent)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadTunnelEvent{
		Target: d.Target,
		Event:  int32(d.Event),
	}

	return proto.Marshal(p)
}
