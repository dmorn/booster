package protocol

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/danielmorandini/booster/protocol/internal"
)

const (
	MessageHello int32 = 0
)

type Header struct {
	proto *internal.Header
}

func (h *Header) ID() int32 {
	return h.proto.Id
}

func (h *Header) ProtocolVersion() string {
	return h.proto.ProtocolVersion
}

func (h *Header) SentAt() (time.Time, error) {
	return ptypes.Timestamp(h.proto.SentAt)
}

func (h *Header) Modules() []string {
	return h.proto.Modules
}

func DecodeHeader(h []byte) (*Header, error) {
	header := new(internal.Header)
	if err := proto.Unmarshal(h, header); err != nil {
		return nil, err
	}

	return &Header {
		proto: header,
	}, nil
}

// HelloHeader creates the hello payload.
func HelloHeader() ([]byte, error) {
	h := &internal.Header {
		Id: MessageHello,
		Modules: []string{ModulePayload},
		SentAt: ptypes.TimestampNow(),
		ProtocolVersion: Version,
	}

	return proto.Marshal(h)
}
