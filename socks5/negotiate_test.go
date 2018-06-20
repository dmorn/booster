/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package socks5_test

import (
	"testing"

	"github.com/danielmorandini/booster/socks5"
)

func TestNegotiate(t *testing.T) {
	s5 := new(socks5.Socks5)
	conn := new(conn)

	var tests = []struct {
		in  []byte
		out []byte
		err bool // should expect negotiation error?
	}{
		{in: []byte{5, 2, 0, 1}, out: []byte{5, 0}, err: false}, // successful response
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
