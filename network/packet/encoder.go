package packet

import (
	"fmt"
	"io"

	"github.com/danielmorandini/booster/protocol"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	e := new(Encoder)
	e.w = w

	return e
}

func (e *Encoder) Encode(p *Packet) error {
	tw := NewTagWriter(e.w)
	me := NewModuleEncoder(e.w)

	// starting tag
	if _, err := tw.Write(protocol.PacketOpeningTag); err != nil {
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
	if _, err := tw.Write(protocol.PacketClosingTag); err != nil {
		return fmt.Errorf("packet: write close tag: %v", err)
	}

	return nil
}

type ModuleEncoder struct {
	w io.Writer
}

func NewModuleEncoder(w io.Writer) *ModuleEncoder {
	e := new(ModuleEncoder)
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
	if _, err := tw.Write(protocol.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// payload size
	buf = make([]byte, 0, 2)
	buf = append(buf, byte(m.size>>8), byte(m.size))
	if _, err := e.w.Write(buf); err != nil {
		return fmt.Errorf("module: unable to write modules size: %v", err)
	}

	// sepatator
	if _, err := tw.Write(protocol.Separator); err != nil {
		return fmt.Errorf("module: write separator: %v", err)
	}

	// encoding
	if _, err := e.w.Write([]byte{m.encoding}); err != nil {
		return fmt.Errorf("module: unable to write encoding type: %v", err)
	}

	// payload open tag
	if _, err := tw.Write(protocol.PayloadOpeningTag); err != nil {
		return fmt.Errorf("module: write payload open tag: %v", err)
	}

	// payload
	if _, err := e.w.Write(m.payload); err != nil {
		return fmt.Errorf("module: unable to write payload: %v", err)
	}

	// payload close tag
	if _, err := tw.Write(protocol.PayloadClosingTag); err != nil {
		return fmt.Errorf("module: write payload close tag: %v", err)
	}

	return nil
}
