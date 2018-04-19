package booster_test

import (
	"testing"
	"reflect"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/protocol"
)

func TestEncode(t *testing.T) {
	b, err := booster.New(1234, 4321)
	if err != nil {
		t.Fatal(err)
	}
	pl := protocol.PayloadHello{
		BPort: "1234",
		PPort: "4312",
	}
	msg := protocol.MessageHello

	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		t.Fatal(err)
	}

	rpl := new(protocol.PayloadHello)
	f := protocol.PayloadDecoders[msg]
	if err = b.Net().Decode(p, protocol.ModulePayload, &rpl, f); err != nil {
		t.Fatal(err)
	}

	// TODO(daniel): to reflect check if DeepEqual
	if !reflect.DeepEqual(*rpl, pl) {
		t.Fatalf("%v != %v, and they should be equal", rpl, pl)
	}
}
