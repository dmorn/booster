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
	e := protocol.NewEncoder()
	v := protocol.PayloadHello{
		BPort: "1234",
		PPort: "4321",
	}
	m := protocol.MessageHello

	// encode that should pass
	_, err := e.Encode(v, m)
	if err != nil {
		t.Fatal(err)
	}

	// encode that should fail
	_, err = e.Encode(v, protocol.MessageNode)
	if err == nil {
		t.Fatalf("encode shuold fail but it did not. Passing %v with message %v - they do not match", v, protocol.MessageNode)
	}

	// encode that should fail
	val := protocol.PayloadNode{}
	_, err = e.Encode(val, m)
	if err == nil {
		t.Fatalf("encode shuold fail but it did not. Passing %v with message %v - they do not match", v, m)
	}
}
