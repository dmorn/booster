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

func TestHelloDecodeEncode(t *testing.T) {
	pp := "1234"
	bp := "4312"

	p, err := protocol.EncodePayloadHello(bp, pp)
	if err != nil {
		t.Fatal(err)
	}

	hp, err := protocol.DecodePayloadHello(p)
	if err != nil {
		t.Fatal(err)
	}

	if hp.BPort != bp {
		t.Fatalf("%v, wanted %v", hp.BPort, bp)
	}
	if hp.PPort != pp {
		t.Fatalf("%v, wanted %v", hp.PPort, pp)
	}
}
