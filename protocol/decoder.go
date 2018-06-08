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
type DecoderFunc func([]byte, interface{}) error

// Implemented default decoders
var PayloadDecoders = map[Message]DecoderFunc{
	MessageHello:      decodeHello,
	MessageConnect:    decodeConnect,
	MessageDisconnect: decodeDisconnect,
	MessageHeartbeat:  decodeHeartbeat,
	MessageMonitor:    decodeMonitor,
	MessageCtrl:       decodeCtrl,

	MessageNetworkStatus: decodeBandwidth,
	MessageNodeStatus:    decodeNode,
	MessageProxyUpdate:   decodeProxyUpdate,
}

var HeaderDecoder = decodeHeader

// Decode takes as input a byte slice and tries to decode it into v.
// f is used for the internal mapping between public and private
// structs used for the data transmission.
//
// v has to be a pointer to a struct.
func Decode(p []byte, v interface{}, f DecoderFunc) error {
	if err := f(p, v); err != nil {
		return err
	}

	return nil
}

// set s into v.
func set(s interface{}, v interface{}) error {
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

func decodeHeader(p []byte, v interface{}) error {
	header := new(internal.Header)
	if err := proto.Unmarshal(p, header); err != nil {
		return err
	}

	t, err := ptypes.Timestamp(header.SentAt)
	if err != nil {
		return err
	}

	return set(&Header{
		ID:              Message(header.Id),
		ProtocolVersion: header.ProtocolVersion,
		SentAt:          t,
		Modules:         header.Modules,
	}, v)
}

func decodeHello(p []byte, v interface{}) error {
	payload := new(internal.PayloadHello)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	return set(&PayloadHello{
		BPort: payload.Bport,
		PPort: payload.Pport,
	}, v)
}

func decodeCtrl(p []byte, v interface{}) error {
	payload := new(internal.PayloadCtrl)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	return set(&PayloadCtrl{
		Operation: Operation(payload.Operation),
	}, v)
}

func decodeBandwidth(p []byte, v interface{}) error {
	payload := new(internal.PayloadBandwidth)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	return set(&PayloadBandwidth{
		Tot:       int(payload.Tot),
		Bandwidth: int(payload.Bandwidth),
		Type:      payload.Type,
	}, v)
}

func decodeMonitor(p []byte, v interface{}) error {
	payload := new(internal.PayloadMonitor)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	features := []Message{}
	for _, v := range payload.Features {
		features = append(features, Message(v))
	}

	return set(&PayloadMonitor{
		Features: features,
	}, v)
}

func decodeConnect(p []byte, v interface{}) error {
	payload := new(internal.PayloadConnect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	return set(&PayloadConnect{
		Target: payload.Target,
	}, v)
}

func decodeDisconnect(p []byte, v interface{}) error {
	payload := new(internal.PayloadDisconnect)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	return set(&PayloadDisconnect{
		ID: payload.Id,
	}, v)
}

func decodeNode(p []byte, v interface{}) error {
	payload := new(internal.PayloadNode)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
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

	return set(&PayloadNode{
		ID:      payload.Id,
		BAddr:   payload.Baddr,
		PAddr:   payload.Paddr,
		Active:  payload.Active,
		Tunnels: ts,
	}, v)
}

func decodeHeartbeat(p []byte, v interface{}) error {
	payload := new(internal.PayloadHeartbeat)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	t, err := ptypes.Timestamp(payload.Ttl)
	if err != nil {
		return err
	}

	return set(&PayloadHeartbeat{
		ID:   payload.Id,
		Hops: int(payload.Hops),
		TTL:  t,
	}, v)
}

func decodeProxyUpdate(p []byte, v interface{}) error {
	payload := new(internal.PayloadProxyUpdate)
	if err := proto.Unmarshal(p, payload); err != nil {
		return err
	}

	return set(&PayloadProxyUpdate{
		Target:    payload.Target,
		Operation: Operation(payload.Operation),
	}, v)
}
