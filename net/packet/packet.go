package packet

import (
	"fmt"

	"github.com/danielmorandini/booster/proto"
)

type Packet struct {
	modules map[string]*Module
}

func New() *Packet {
	return &Packet{
		modules: make(map[string]*Module),
	}
}

func (p *Packet) AddModule(id string, payload []byte, encoding uint8) (*Module, error) {
	if _, ok := p.modules[id]; ok {
		return nil, fmt.Errorf("packet: module [%v] already present", id)
	}

	m, err := NewModule(id, payload, encoding)
	if err != nil {
		return nil, err
	}

	p.modules[id] = m
	return m, nil
}

func (p *Packet) DelModule(id string) error {
	delete(p.modules, id)

	return nil
}

func (p *Packet) Module(id string) (*Module, error) {
	m, ok := p.modules[id]
	if !ok {
		return nil, fmt.Errorf("packet: module [%v] not found", id)
	}

	return m, nil
}

type Module struct {
	id       string
	size     uint16
	encoding uint8
	payload  []byte
}

func NewModule(id string, payload []byte, encoding uint8) (*Module, error) {
	if len([]byte(id)) != 2 {
		return nil, fmt.Errorf("module: id must be a 2 letters identifier: example: HE. Found %v", id)
	}

	size := len(payload)
	if size < 1 || size > 0xffff {
		return nil, fmt.Errorf("module: payload size out of bounds: %v", size)
	}

	return &Module{
		id:       string(id),
		size:     uint16(size),
		encoding: encoding,
		payload:  payload,
	}, nil
}

func (m *Module) ID() string {
	return m.id
}

func (m *Module) Payload() []byte {
	return m.payload
}

func (m *Module) Encoding() string {
	switch m.encoding {
	case proto.EncodingProtobuf:
		return "protobuf"
	default:
		return "undefined"
	}
}
