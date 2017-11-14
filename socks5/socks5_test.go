package socks5_test

import (
	"bytes"
	"testing"

	"github.com/danielmorandini/booster/socks5"
)

type conn struct {
	buf bytes.Buffer
}

func (c *conn) Read(p []byte) (int, error) {
	return c.buf.Read(p)
}

func (c *conn) Write(p []byte) (int, error) {
	return c.buf.Write(p)
}

func (c *conn) Close() {}

func TestReadAddress(t *testing.T) {
	conn := new(conn)

	var tests = []struct {
		in  []byte
		out string
		err bool
	}{
		{in: []byte{1, 93, 184, 216, 34, 1, 187},
			out: "93.184.216.34:443",
			err: false}, // ipv4

		{in: []byte{4, 42, 3, 176, 192, 0, 3, 0, 208, 0, 0, 0, 0, 72, 136, 160, 1, 1, 187},
			out: "[2a03:b0c0:3:d0::4888:a001]:443",
			err: false}, // ipv6

		{in: []byte{3, 21, 111, 117, 116, 108, 111, 111, 107, 46, 111, 102, 102, 105, 99, 101, 51, 54, 53, 46, 99, 111, 109, 1, 187},
			out: "outlook.office365.com:443",
			err: false}, // FQDN

		{in: []byte{0, 93, 184, 216, 34, 1, 187},
			out: "",
			err: true}, // wrong address type

		{in: []byte{5, 93, 184, 216, 34, 1, 187},
			out: "",
			err: true}, // wrong address type
	}

	for _, test := range tests {
		if _, err := conn.Write(test.in); err != nil {
			t.Fatal(err)
		}

		s, err := socks5.ReadAddress(conn)
		if err != nil {
			// only fail if not expecting an error
			if !test.err {
				t.Fatal(err)
			} else {
				return
			}
		}

		t.Log("Address Read: " + s)

		if s != test.out {
			t.Fatal(err)
		}
	}
}
