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

	// encoding
	if _, err := e.w.Write([]byte{p.M.Encoding}); err != nil {
		return fmt.Errorf("packet: write encoding: %v", err)
	}
	// sepatator
	if _, err := tw.Write(e.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// compression
	if _, err := e.w.Write([]byte{p.M.Compression}); err != nil {
		return fmt.Errorf("packet: write compression: %v", err)
	}
	// sepatator
	if _, err := tw.Write(e.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// encryption
	if _, err := e.w.Write([]byte{p.M.Encryption}); err != nil {
		return fmt.Errorf("packet: write encprytion: %v", err)
	}
	// sepatator
	if _, err := tw.Write(e.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// modules size
	mn := len(p.modules)
	if mn < 1 || mn > 0xff {
		return fmt.Errorf("packet: too many modules")
	}
	if _, err := e.w.Write([]byte{byte(mn)}); err != nil {
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

	// module opening tag
	if _, err := tw.Write(e.ModuleOpeningTag); err != nil {
		return fmt.Errorf("module: write module open tag: %v", err)
	}

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

	// payload
	if _, err := e.w.Write(m.payload); err != nil {
		return fmt.Errorf("module: unable to write payload: %v", err)
	}

	// module closing tag
	if _, err := tw.Write(e.ModuleClosingTag); err != nil {
		return fmt.Errorf("module: write module close tag: %v", err)
	}

	return nil
}
