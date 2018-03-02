package protocol_test

import (
	"testing"

	"github.com/danielmorandini/booster/protocol"
)

func TestHelloDecodeEncode(t *testing.T) {
	pp := "1234"
	bp := "4312"

	p, err := protocol.HelloPayload(bp, pp)
	if err != nil {
		t.Fatal(err)
	}

	hp, err := protocol.DecodePayloadHello(p)
	if err != nil {
		t.Fatal(err)
	}

	if hp.BPort != bp {
		t.Fatalf("%v, wanted %v", hp.BPort, bp)
	}
	if hp.PPort != pp {
		t.Fatalf("%v, wanted %v", hp.PPort, pp)
	}
}
