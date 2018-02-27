package protocol

type Packet interface {
	Module(id string) (Module, error)
}

type Module interface {
	ID() string
	Payload() []byte
	Encoding() string
}
