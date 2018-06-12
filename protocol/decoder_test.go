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
	"fmt"
	"reflect"
	"testing"

	"github.com/danielmorandini/booster/protocol"
)

func mockDecodeHello(p []byte, v interface{}) error {
	payload := &protocol.PayloadHello{
		BPort: "1234",
		PPort: "4321",
	}
	val := reflect.ValueOf(payload)

	ptr := reflect.ValueOf(v).Elem()
	if !ptr.CanSet() {
		return fmt.Errorf("unable to set to %v", ptr)
	}

	if ptr.Type() != val.Type() {
		return fmt.Errorf("types differ: %T != %T", ptr, val)
	}

	ptr.Set(val)
	return nil
}

func TestDecode(t *testing.T) {
	p := []byte{}
	m := protocol.MessageHello
	f := mockDecodeHello

	// decode that shuold pass
	v := new(protocol.PayloadHello)
	if err := protocol.Decode(p, &v, f); err != nil {
		t.Fatal(err)
	}

	if v.BPort != "1234" {
		t.Fatalf("unexpected BPort: found %v, wanted 1234", v.BPort)
	}

	// decode that should fail
	fd := new(protocol.PayloadNode)
	if err := protocol.Decode(p, &fd, f); err == nil {
		t.Fatalf("decode should fail but it did not. Decoding %v with message %v", fd, m)
	}

	// decode that should fail
	if err := protocol.Decode(p, v, f); err == nil {
		t.Fatalf("decode shuold fail but it did not. Passed %v as value", v)
	}
}
