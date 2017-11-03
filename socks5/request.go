package socks5

import "fmt"

type Request struct {
	Unmarshaler

	Ver      uint8
	Cmd      uint8
	AddrType uint8
	Addr     []uint8
	DstPort  []uint8
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
	expl := 5 // expected p length
	if len(p) < expl {
		return fmt.Errorf("unexpected input length. found %v", len(p))
	}
	v := p[0]    // version
	cmd := p[1]  // command
	atyp := p[3] // address type

	// destination address
	var addr []byte
	var addrl int // address length
	asi := 4      // address start index

	switch atyp {
	case AddrTypeIPV4:
		addrl = 4
	case AddrTypeIPV6:
		addrl = 16
	case AddrTypeDomainName:
		addrl = int(p[4])
		asi = 5
	}

	expl = 6 + addrl
	if len(p) < expl {
		return fmt.Errorf("unexpected input length. found %v", len(p))
	}

	addr = make([]uint8, addrl)
	copy(addr, p[asi:(asi+addrl)])
	if len(addr) == 0 {
		return fmt.Errorf("unable to parse destination address")
	}

	// destination port
	psi := asi + len(addr) // port starting index
	dstPort := p[psi:(psi + 2)]
	if len(dstPort) == 0 {
		return fmt.Errorf("unable to parse destination port")
	}

	r.Ver = v
	r.Cmd = cmd
	r.AddrType = atyp
	r.Addr = addr
	r.DstPort = dstPort

	return nil
}
