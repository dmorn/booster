package socks5_test

import (
	"testing"

	"github.com/danielmorandini/booster-network/socks5"
)

func TestNegotiate(t *testing.T) {
	s5 := new(socks5.Socks5)
	conn := new(conn)

	var tests = []struct {
		in  []byte
		out []byte
		err bool // should expect negotiation error?
	}{
		{in: []byte{5, 2, 0, 1}, out: []byte{5, 0}, err: false}, // successfull response
		{in: []byte{5, 1, 1}, out: []byte{5, 0xff}, err: false}, // command not supported
		{in: []byte{4}, out: []byte{}, err: true},               // wrong version
		{in: []byte{5, 0, 1}, out: []byte{}, err: true},         // wrong methods number
	}

	for _, test := range tests {
		if _, err := conn.Write(test.in); err != nil {
			t.Fatal(err)
		}

		if err := s5.Negotiate(conn); err != nil {
			// only fail if not expecting any error
			if !test.err {
				t.Fatal(err)
			} else {
				return
			}
		}

		buf := make([]byte, 2)
		if _, err := conn.Read(buf); err != nil {
			t.Fatal(err)
		}

		for i, v := range buf {
			if v != test.out[i] {
				t.Fatalf("unexpected result. Wanted %v, found %v", test.out, buf)
			}
		}

	}
}
