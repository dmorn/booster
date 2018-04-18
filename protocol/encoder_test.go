package protocol_test

import (
	"testing"

	"github.com/danielmorandini/booster/protocol"
)

func TestEncode(t *testing.T) {
	e := protocol.NewEncoder()
	v := protocol.PayloadHello{
		BPort: "1234",
		PPort: "4321",
	}
	m := protocol.MessageHello

	// encode that should pass
	_, err := e.Encode(v, m)
	if err != nil {
		t.Fatal(err)
	}

	// encode that should fail
	_, err = e.Encode(v, protocol.MessageNode)
	if err == nil {
		t.Fatalf("encode shuold fail but it did not. Passing %v with message %v - they do not match", v, protocol.MessageNode)
	}

	// encode that should fail
	val := protocol.PayloadNode{}
	_, err = e.Encode(val, m)
	if err == nil {
		t.Fatalf("encode shuold fail but it did not. Passing %v with message %v - they do not match", v, m)
	}
}
