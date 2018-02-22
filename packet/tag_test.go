package packet_test

import (
	"io"
	"strings"
	"testing"

	"github.com/danielmorandini/booster-network/packet"
)

func TestTagRead(t *testing.T) {
	r := strings.NewReader(">")
	tr := packet.NewTagReader(r, ">")

	// test with bigger buffer
	buf := make([]byte, 4)
	n, err := tr.Read(buf)
	if err != io.EOF {
		t.Fatal(err)
	}

	if n != 1 {
		t.Fatalf("%v, wanted 1", n)
	}

	if buf[0] != '>' {
		t.Fatalf("%v, wanted >", buf[0])
	}

	r = strings.NewReader(">")
	tr = packet.NewTagReader(r, ">")

	// test with smaller buffer
	buf = buf[:1]
	n, err = tr.Read(buf)
	if err != io.EOF {
		t.Fatal(err)
	}

	if n != 1 {
		t.Fatalf("%v, wanted 1", n)
	}

	if buf[0] != '>' {
		t.Fatalf("%v, wanted >", buf[0])
	}

	// test with wrong tags
	r = strings.NewReader("-")
	tr = packet.NewTagReader(r, ">")

	n, err = tr.Read(buf)
	if err == io.EOF {
		t.Fatal(err)
	}
}
