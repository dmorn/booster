package socks5

import "fmt"

type Response struct {
	Marshaler

	Ver     uint8
	Resp    uint8 // reponse code
	BndAddr *Addr
}

func NewResponse(bdnAddr *Addr, code uint8) (*Response, error) {
	res := new(Response)
	res.Ver = Version5
	res.Resp = code
	res.BndAddr = bdnAddr

	return res, nil
}

// Marshal converts r into []byte.
// Output format:
//
// +----+-----+-------+------+----------+----------+
// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// +----+-----+-------+------+----------+----------+
// | 1  |  1  | X'00' |  1   | Variable |    2     |
// +----+-----+-------+------+----------+----------+
// numers represent field size
//
func (r *Response) Marshal() ([]byte, error) {
	// Format the address
	var addrType uint8
	var addrBody []byte
	var addrPort uint16
	addr := r.BndAddr

	switch {
	case addr == nil:
		addrType = AddrTypeIPV4
		addrBody = []byte{0, 0, 0, 0}
		addrPort = 0

	case addr.FQDN != "":
		addrType = AddrTypeFQDN
		addrBody = append([]byte{byte(len(addr.FQDN))}, addr.FQDN...)
		addrPort = uint16(addr.Port)

	case addr.IP.To4() != nil:
		addrType = AddrTypeIPV4
		addrBody = []byte(addr.IP.To4())
		addrPort = uint16(addr.Port)

	case addr.IP.To16() != nil:
		addrType = AddrTypeIPV6
		addrBody = []byte(addr.IP.To16())
		addrPort = uint16(addr.Port)

	default:
		return nil, fmt.Errorf("Failed to format address: %v", addr)
	}

	// Format the message
	buf := make([]byte, 6+len(addrBody))
	buf[0] = Version5
	buf[1] = r.Resp
	buf[2] = FieldReserved
	buf[3] = addrType
	copy(buf[4:], addrBody)
	buf[4+len(addrBody)] = byte(addrPort >> 8)
	buf[4+len(addrBody)+1] = byte(addrPort & 0xff)

	return buf, nil
}
