package protocol

import (
	"time"

	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

const (
	MessageHello int32 = 0
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

// HelloHeader creates the hello payload.
func HelloHeader() ([]byte, error) {
	h := &internal.Header{
		Id:              MessageHello,
		Modules:         []string{ModulePayload},
		SentAt:          ptypes.TimestampNow(),
		ProtocolVersion: Version,
	}

	return proto.Marshal(h)
}
