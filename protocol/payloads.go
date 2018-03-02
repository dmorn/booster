package protocol

import (
	"github.com/danielmorandini/booster/protocol/internal"
	"github.com/golang/protobuf/proto"
)

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

func HelloPayload(bport, pport string) ([]byte, error) {
	p := &internal.PayloadHello{
		Pport: pport,
		Bport: bport,
	}

	return proto.Marshal(p)
}
