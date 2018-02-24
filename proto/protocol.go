package proto

type Packet interface {
	Module(id string) (Module, error)
}

type Module interface {
	ID() string
	Payload() []byte
	Encoding() string
}

type Conn interface {
	Consume() (<-chan Packet, error)
	Send(Packet) error
	Close() error
	Err() error
}

