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
var PayloadEncoders = map[Message]EncoderFunc{
	MessageHello:      encodeHello,
	MessageConnect:    encodeConnect,
	MessageDisconnect: encodeDisconnect,
	MessageHeartbeat:  encodeHeartbeat,
	MessageMonitor:    encodeMonitor,
	MessageCtrl:       encodeCtrl,

	MessageNodeStatus:         encodeNode,
	MessageNetworkUsageUpdate: encodeBandwidth,
	MessageProxyUpdate:        encodeProxyUpdate,
	MessageNetworkUpdate:      encodeNetworkUpdate,
}

// HeaderEncoder is the default function used to encode the headers.
var HeaderEncoder = encodeHeader

// Encoder tries to encode v using f.
//
// v has to be a value, not a pointer (in fact we don't want v to be
// modified by this function in any way).
// When encoding an header using the default HeaderEncoders, v has to
// be a Message, which will be used to choose how to build the header
// using a default configuration, such as the encoding will be set to
// protobuf, the modules field will contain the payload module, etc.
// When encoding custom packets its betters to pass a custom EncoderFunc
// as parameter.
func Encode(v interface{}, f EncoderFunc) ([]byte, error) {
	return f(v)
}

func conversionFail(v interface{}) error {
	return fmt.Errorf("protocol: encode error: unable to make type assertion: %v is of unexpected type %v", v, reflect.TypeOf(v))
}

func newHeader(id Message) *internal.Header {
	return &internal.Header{
		Id:              int32(id),
		Modules:         []string{string(ModulePayload)},
		SentAt:          ptypes.TimestampNow(),
		ProtocolVersion: Version,
	}
}

func encodeHeader(v interface{}) ([]byte, error) {
	d, ok := v.(Message)
	if !ok {
		return nil, conversionFail(v)
	}

	h := newHeader(d)

	if d == MessageNotify {
		h.Modules = []string{}
	}

	return proto.Marshal(h)
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
		NodeID:    d.NodeID,
		Tot:       int64(d.Tot),
		Bandwidth: int64(d.Bandwidth),
		Type:      d.Type,
	}

	return proto.Marshal(p)
}

func encodeMonitor(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadMonitor)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadMonitor{
		Feature: int32(d.Feature),
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

func nodeToInternal(d *PayloadNode) *internal.PayloadNode {
	ts := []*internal.PayloadNode_Tunnel{}
	for _, t := range d.Tunnels {
		tunnel := &internal.PayloadNode_Tunnel{
			Id:     t.ID,
			Target: t.Target,
			Copies: int32(t.Copies),
		}

		ts = append(ts, tunnel)
	}

	return &internal.PayloadNode{
		Id:      d.ID,
		Baddr:   d.BAddr,
		Paddr:   d.PAddr,
		Active:  d.Active,
		Tunnels: ts,
	}
}

func encodeNode(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadNode)
	if !ok {
		return nil, conversionFail(v)
	}

	return proto.Marshal(nodeToInternal(&d))
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

func encodeProxyUpdate(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadProxyUpdate)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadProxyUpdate{
		NodeID:    d.NodeID,
		Target:    d.Target,
		Operation: int32(d.Operation),
	}

	return proto.Marshal(p)
}

func encodeNetworkUpdate(v interface{}) ([]byte, error) {
	d, ok := v.(PayloadNetworkUpdate)
	if !ok {
		return nil, conversionFail(v)
	}

	p := &internal.PayloadNetworkUpdate{
		NodeID:     d.NodeID,
		RemoteNode: nodeToInternal(d.RemoteNode),
		Operation:  int32(d.Operation),
	}

	return proto.Marshal(p)
}
