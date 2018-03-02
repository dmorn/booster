package packet_test

import (
	"io"
	"testing"

	"github.com/danielmorandini/booster/network/packet"
	"github.com/danielmorandini/booster/protocol"
)

var tagset packet.TagSet = packet.TagSet{
	PacketOpeningTag:  protocol.PacketOpeningTag,
	PacketClosingTag:  protocol.PacketClosingTag,
	PayloadClosingTag: protocol.PayloadClosingTag,
	Separator:         protocol.Separator,
}

func TestAddModule(t *testing.T) {
	p := packet.New()
	pl := []byte("booster")
	id := protocol.ModuleHeader

	// try to add the header module
	m, err := p.AddModule(id, pl, 0)
	if err != nil {
		t.Fatal(err)
	}

	hm, err := p.Module(id)
	if err != nil {
		t.Fatal(err)
	}

	if hm.ID() != m.ID() {
		t.Fatalf("wanted %v, found %v", m.ID(), hm.ID())
	}

	// try to add a custom module
	id = "fo"
	m, err = p.AddModule(id, pl, 0)
	if err != nil {
		t.Fatal(err)
	}

	hm, err = p.Module(id)
	if err != nil {
		t.Fatal(err)
	}

	if hm.ID() != m.ID() {
		t.Fatalf("wanted %v, found %v", m.ID(), hm.ID())
	}

	id = "fk"
	if _, err = p.Module(id); err == nil {
		t.Fatalf("unexpected module [%v] found", id)
	}
}

func TestEncodeDecode(t *testing.T) {
	p := packet.New()
	pl := []byte("header")
	ppl := []byte("payload")
	hid := protocol.ModuleHeader
	pid := protocol.ModulePayload

	m, err := p.AddModule(hid, pl, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.AddModule(pid, ppl, 0)
	if err != nil {
		t.Fatal(err)
	}

	r, w := io.Pipe()
	pe := packet.NewEncoder(w, tagset)
	pd := packet.NewDecoder(r, tagset)

	go func() {
		if err = pe.Encode(p); err != nil {
			t.Fatal(err)
		}
	}()

	pr := packet.New() // packet read
	if err = pd.Decode(pr); err != nil {
		t.Fatal(err)
	}

	// check that the received packet also has the header module
	hm, err := pr.Module(protocol.ModuleHeader)
	if err != nil {
		t.Fatal(err)
	}

	if hm.ID() != m.ID() {
		t.Fatalf("wanted %v, found %v", m.ID(), hm.ID())
	}
}
