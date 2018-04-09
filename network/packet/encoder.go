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

package packet

import (
	"fmt"
	"io"
)

type Encoder struct {
	TagSet
	w io.Writer
}

func NewEncoder(w io.Writer, t TagSet) *Encoder {
	e := new(Encoder)
	e.TagSet = t
	e.w = w

	return e
}

func (e *Encoder) Encode(p *Packet) error {
	tw := NewTagWriter(e.w)
	me := NewModuleEncoder(e.w, e.TagSet)

	// starting tag
	if _, err := tw.Write(e.PacketOpeningTag); err != nil {
		return fmt.Errorf("packet: write open tag: %v", err)
	}

	// modules size
	mn := len(p.modules)
	if mn < 1 || mn > 0xffff {
		return fmt.Errorf("packet: too many modules")
	}

	buf := make([]byte, 0, 2)
	buf = append(buf, byte(mn>>8), byte(mn))
	if _, err := e.w.Write(buf); err != nil {
		return fmt.Errorf("packet: unable to write modules size: %v", err)
	}

	// modules
	for _, m := range p.modules {
		if err := me.Encode(m); err != nil {
			return err
		}
	}

	// closing tag
	if _, err := tw.Write(e.PacketClosingTag); err != nil {
		return fmt.Errorf("packet: write close tag: %v", err)
	}

	return nil
}

type ModuleEncoder struct {
	TagSet
	w io.Writer
}

func NewModuleEncoder(w io.Writer, t TagSet) *ModuleEncoder {
	e := new(ModuleEncoder)
	e.TagSet = t
	e.w = w

	return e
}

func (e *ModuleEncoder) Encode(m *Module) error {
	tw := NewTagWriter(e.w)

	// module id
	buf := []byte(m.id)
	if len(buf) != 2 {
		return fmt.Errorf("module: ID must be a 2 letters identifier, found %v", m.ID())
	}

	if _, err := e.w.Write(buf); err != nil {
		return fmt.Errorf("module: unable to write module id: %v", err)
	}

	// sepatator
	if _, err := tw.Write(e.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// payload size
	buf = make([]byte, 0, 2)
	buf = append(buf, byte(m.size>>8), byte(m.size))
	if _, err := e.w.Write(buf); err != nil {
		return fmt.Errorf("module: unable to write modules size: %v", err)
	}

	// sepatator
	if _, err := tw.Write(e.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// encoding
	if _, err := e.w.Write([]byte{m.encoding}); err != nil {
		return fmt.Errorf("module: unable to write encoding type: %v", err)
	}

	// payload open tag
	if _, err := tw.Write(e.PayloadOpeningTag); err != nil {
		return fmt.Errorf("module: write payload open tag: %v", err)
	}

	// payload
	if _, err := e.w.Write(m.payload); err != nil {
		return fmt.Errorf("module: unable to write payload: %v", err)
	}

	// payload close tag
	if _, err := tw.Write(e.PayloadClosingTag); err != nil {
		return fmt.Errorf("module: write payload close tag: %v", err)
	}

	return nil
}
