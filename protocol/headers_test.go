package protocol_test

import (
	"testing"

	"github.com/danielmorandini/booster/protocol"
)

func TestEncodeDecode(t *testing.T) {
	p, err := protocol.HelloHeader()
	if err != nil {
		t.Fatal(err)
	}

	h, err := protocol.DecodeHeader(p)
	if err != nil {
		t.Fatal(err)
	}

	if h.ID != protocol.MessageHello {
		t.Fatalf("%v, wanted %v", h.ID, protocol.MessageHello)
	}

	if h.ProtocolVersion != protocol.Version {
		t.Fatalf("%v, wanted %v", h.ProtocolVersion, protocol.Version)
	}
}
