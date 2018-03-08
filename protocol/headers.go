package protocol

import (
	"time"

	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

type Header struct {
	ID              int32
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
		ID:              header.Id,
		ProtocolVersion: header.ProtocolVersion,
		SentAt:          t,
		Modules:         header.Modules,
	}, nil
}

func newHP(id int32) *internal.Header {
	return &internal.Header{
		Id:              id,
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
