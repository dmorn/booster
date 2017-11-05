package socks5_test

import (
	"testing"

	"github.com/danielmorandini/booster/socks5"
)

func TestString(t *testing.T) {
	var tests = []struct {
		input []byte
		out   string
	}{
		{input: []byte{1, 93, 184, 216, 34, 0, 80}, out: "93.184.216.34:80"},
	}

	for _, test := range tests {
		addr := new(socks5.Addr)
		if err := addr.Unmarshal(test.input); err != nil {
			t.Fatal(err)
		}

		if addr.String() != test.out {
			t.Fatalf("expected %v, found %v", test.out, addr.String())
		}
	}
}
