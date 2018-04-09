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

package packet_test

import (
	"io"
	"strings"
	"testing"

	"github.com/danielmorandini/booster/network/packet"
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
