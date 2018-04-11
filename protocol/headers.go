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
	"time"

	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type Header struct {
	ID              Message
	ProtocolVersion string
	SentAt          time.Time
	Modules         []string
}

func (h *Header) HasModule(m string) bool {
	for _, v := range h.Modules {
		if v == m {
			return true
		}
	}
	return false
}

// DecodeHeader decodes the given header and returns it.
func DecodeHeader(h []byte) (*Header, error) {
	header := new(internal.Header)
	if err := proto.Unmarshal(h, header); err != nil {
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

func newHP(id Message) *internal.Header {
	return &internal.Header{
		Id:              int32(id),
		Modules:         []string{ModulePayload},
		SentAt:          ptypes.TimestampNow(),
		ProtocolVersion: Version,
	}
}

func HelloHeader() ([]byte, error) {
	h := newHP(MessageHello)
	return proto.Marshal(h)
}

func ConnectHeader() ([]byte, error) {
	h := newHP(MessageConnect)
	return proto.Marshal(h)
}

func DisconnectHeader() ([]byte, error) {
	h := newHP(MessageDisconnect)
	return proto.Marshal(h)
}

func NodeHeader() ([]byte, error) {
	h := newHP(MessageNode)
	return proto.Marshal(h)
}

func HeartbeatHeader() ([]byte, error) {
	h := newHP(MessageHeartbeat)
	return proto.Marshal(h)
}

func TunnelEventHeader() ([]byte, error) {
	h := newHP(MessageTunnel)
	return proto.Marshal(h)
}

func TunnelNotifyHeader() ([]byte, error) {
	h := newHP(MessageNotify)
	h.Modules = []string{}
	return proto.Marshal(h)
}

func InspectHeader() ([]byte, error) {
	h := newHP(MessageInspect)
	return proto.Marshal(h)
}

func BandwidthHeader() ([]byte, error) {
	h := newHP(MessageBandwidth)
	return proto.Marshal(h)
}

func CtrlHeader() ([]byte, error) {
	h := newHP(MessageCtrl)
	return proto.Marshal(h)
}
