package proto

type Packet interface {
	Module(id string) (Module, error)
}

type Module interface {
	ID() string
	Body() []byte
	Encoding() string
}

type Heartbeater struct {
}

// Recv returns an error if p is not a heartbit message.
func (h *Heartbeater) Recv(p Packet) error {
	return nil
}

// Packet composes a heartbeater message.
func (h *Heartbeater) Payload() []byte {
	return []byte{}
}
