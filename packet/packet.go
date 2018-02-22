package packet

import (
	"io"
)

type Packet struct {
	HeaderSize uint16
	Header []byte

	PayloadSize uint16
	Payload []byte
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	d := new(Decoder)
	if _, ok := r.(io.ByteReader); !ok {
		r = bufio.NewReader(r)
	}

	d.r = r

	return d
}

func (d *Decoder) Decode(p *Packet) error {
	hs, err := d.r.ReadByte()
	if err != nil {
		return fmt.Errorf("packet: read header failed: %v", err)
	}
	if hs != 'h' {
		return fmt.Errorf("packet: unrecognised header start: %v, wanted h", hs)
	}

}
