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

package booster_test

import (
	"reflect"
	"testing"

	"github.com/danielmorandini/booster/booster"
	"github.com/danielmorandini/booster/protocol"
)

func TestEncode(t *testing.T) {
	b, err := booster.New(1234, 4321)
	if err != nil {
		t.Fatal(err)
	}
	pl := protocol.PayloadHello{
		BPort: "1234",
		PPort: "4312",
	}
	msg := protocol.MessageHello

	p, err := b.Net().Encode(pl, msg)
	if err != nil {
		t.Fatal(err)
	}

	rpl := new(protocol.PayloadHello)
	f := protocol.PayloadDecoders[msg]
	if err = b.Net().Decode(p, protocol.ModulePayload, &rpl, f); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(*rpl, pl) {
		t.Fatalf("%v != %v, and they should be equal", rpl, pl)
	}
}
