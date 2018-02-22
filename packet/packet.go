package packet

import (
	"fmt"
)

const (
	EncodingProto uint8 = 1
)

const (
	ModuleHeader string = "HE"
	ModulePayload = "PA"
)

type Packet struct {
	modules map[string]*Module
}

func New() *Packet {
	return &Packet {
		modules: make(map[string]*Module),
	}
}

type Module struct {
	ID string
	Size uint16
	Encoding uint8
	Payload []byte
}

func (p *Packet) Header() (*Module, error) {
	m, ok := p.modules[ModuleHeader]
	if !ok {
		return nil, fmt.Errorf("packet: no header module")
	}

	return m, nil
}

