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

// DecoderFunc defines how a decoder should behave.
type DecoderFunc func([]byte) (interface{}, error)

// Implemented default decoders
var PayloadDecoders = map[Message]DecoderFunc{
	MessageHello:      decodeHello,
	MessageCtrl:       decodeCtrl,
	MessageBandwidth:  decodeBandwidth,
	MessageMonitor:    decodeMonitor,
	MessageConnect:    decodeConnect,
	MessageDisconnect: decodeDisconnect,
	MessageNode:       decodeNode,
	MessageHeartbeat:  decodeHeartbeat,
	MessageTunnel:     decodeTunnelEvent,
}

var HeaderDecoder = decodeHeader

// Decode takes as input a byte slice and tries to decode it into v.
// f is used for the internal mapping between public and private
// structs used for the data transmission.
//
// v has to be a pointer to a struct.
func Decode(p []byte, v interface{}, f DecoderFunc) error {
	s, err := f(p)
	if err != nil {
		return fmt.Errorf("protocol: decode error: %v", err)
	}

	// reflect the actual value decoded
	val := reflect.ValueOf(s)

	// reflect the actual value of the interface provided. Need to retrieve its pointer
	// in order to be able to set to it
	ptr := reflect.ValueOf(v).Elem()
	if !ptr.CanSet() {
		return fmt.Errorf("protocol: unable to set on value, pass pointer to struct instead")
	}

	// Copy contents of the decoded payload into the parameter
	// check if we're talking about the same thing
	if ptr.Type() != val.Type() {
		return fmt.Errorf("protocol: decode error: trying to reflect %v into %v, which is illegal", val.Type(), ptr.Type())
	}

	ptr.Set(val)
	return nil
}

// TODO: probably this whole boilerplate code below could be replaced
// using reflection.

func decodeHeader(p []byte) (interface{}, error) {
	header := new(internal.Header)
	if err := proto.Unmarshal(p, header); err != nil {
		return nil, err
	}

	t, err := ptypes.Timestamp(header.SentAt)
	if err != nil {
		return nil, err
	}

	return &Header{
		ID:              Message(header.Id),
		ProtocolVersion: header.ProtocolVersion,
		SentAt:          t,
		Modules:         header.Modules,
	}, nil
}

func decodeHello(p []byte) (interface{}, error) {
	payload := new(internal.PayloadHello)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadHello{
		BPort: payload.Bport,
		PPort: payload.Pport,
	}, nil
}

func decodeCtrl(p []byte) (interface{}, error) {
	payload := new(internal.PayloadCtrl)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadCtrl{
		Operation: Operation(payload.Operation),
	}, nil
}

func decodeBandwidth(p []byte) (interface{}, error) {
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

func decodeMonitor(p []byte) (interface{}, error) {
	payload := new(internal.PayloadMonitor)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	features := []Message{}
	for _, v := range payload.Features {
		features = append(features, Message(v))
	}

	return &PayloadMonitor{
		Features: features,
	}, nil
}

func decodeConnect(p []byte) (interface{}, error) {
	payload := new(internal.PayloadConnect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadConnect{
		Target: payload.Target,
	}, nil
}

func decodeDisconnect(p []byte) (interface{}, error) {
	payload := new(internal.PayloadDisconnect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadDisconnect{
		ID: payload.Id,
	}, nil
}

func decodeNode(p []byte) (interface{}, error) {
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

func decodeHeartbeat(p []byte) (interface{}, error) {
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

func decodeTunnelEvent(p []byte) (interface{}, error) {
	payload := new(internal.PayloadTunnelEvent)
	if err := proto.Unmarshal(p, payload); err != nil {
		return nil, err
	}

	return &PayloadTunnelEvent{
		Target: payload.Target,
		Event:  int(payload.Event),
	}, nil
}
