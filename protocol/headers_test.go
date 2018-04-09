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

package protocol_test

import (
	"testing"

	"github.com/danielmorandini/booster/protocol"
)

func TestEncodeDecode(t *testing.T) {
	p, err := protocol.HelloHeader()
	if err != nil {
		t.Fatal(err)
	}

	h, err := protocol.DecodeHeader(p)
	if err != nil {
		t.Fatal(err)
	}

	if h.ID != protocol.MessageHello {
		t.Fatalf("%v, wanted %v", h.ID, protocol.MessageHello)
	}

	if h.ProtocolVersion != protocol.Version {
		t.Fatalf("%v, wanted %v", h.ProtocolVersion, protocol.Version)
	}
}
