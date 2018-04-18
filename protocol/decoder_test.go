package protocol_test

import (
	"testing"

	"github.com/danielmorandini/booster/protocol"
)

func mockDecodeHello(p []byte) (interface{}, error) {
	return &protocol.PayloadHello{
		BPort: "1234",
		PPort: "4321",
	}, nil
}

func TestDecode(t *testing.T) {
	d := protocol.NewDecoder()
	p := []byte{}
	m := protocol.MessageHello
	// add a fake decoder
	d.Decoders[m] = mockDecodeHello

	// decode that shuold pass
	v := new(protocol.PayloadHello)
	if err := d.Decode(p, &v, m); err != nil {
		t.Fatal(err)
	}

	if v.BPort != "1234" {
		t.Fatalf("unexpected BPort: found %v, wanted 1234", v.BPort)
	}

	// decode that should fail
	fd := new(protocol.PayloadNode)
	if err := d.Decode(p, &fd, m); err == nil {
		t.Fatalf("decode should fail but it did not. Decoding %v with message %v", fd, m)
	}

	// decode that should fail
	if err := d.Decode(p, v, m); err == nil {
		t.Fatalf("decode shuold fail but it did not. Passed %v as value", v)
	}
}
