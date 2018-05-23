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

func TestEncode(t *testing.T) {
	v := protocol.PayloadHello{
		BPort: "1234",
		PPort: "4321",
	}
	m := protocol.MessageHello
	f := protocol.PayloadEncoders[m]

	// encode that should pass
	_, err := protocol.Encode(v, f)
	if err != nil {
		t.Fatal(err)
	}

	// encode that should fail
	f = protocol.PayloadEncoders[protocol.MessageNodeStatus]
	_, err = protocol.Encode(v, f)
	if err == nil {
		t.Fatalf("encode shuold fail but it did not. Passing %v with message %v - they do not match", v, protocol.MessageNodeStatus)
	}

	// encode that should fail
	val := protocol.PayloadNode{}
	f = protocol.PayloadEncoders[m]
	_, err = protocol.Encode(val, f)
	if err == nil {
		t.Fatalf("encode shuold fail but it did not. Passing %v with message %v - they do not match", v, m)
	}

	// encode that should fail
	ptr := new(protocol.PayloadNode)
	if _, err = protocol.Encode(ptr, f); err == nil {
		t.Fatalf("encode should fail but it did not. Passing pointer value %v", &v)
	}
}
