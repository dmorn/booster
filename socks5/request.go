package socks5

import (
	"fmt"
)

type Request struct {
	Unmarshaler

	Ver      uint8
	Cmd      uint8
	DestAddr *Addr
}

// Unmarshal fill r with data contained in p.
// expected input format:
//
// +----+-----+-------+------+----------+----------+
// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
// +----+-----+-------+------+----------+----------+
// | 1  |  1  | X'00' |  1   | Variable |    2     |
// +----+-----+-------+------+----------+----------+
// numers represent field size
//
func (r *Request) Unmarshal(p []byte) error {
	expl := 3 // expected p length
	if len(p) < expl {
		return fmt.Errorf("unexpected input length. found %v", len(p))
	}
	v := p[0]   // version
	cmd := p[1] // command

	// read address/port
	addr := new(Addr)
	if err := addr.Unmarshal(p[3:]); err != nil {
		return err
	}

	fmt.Printf("Address translated: %v\n", addr.String())

	r.Ver = v
	r.Cmd = cmd
	r.DestAddr = addr

	return nil
}
