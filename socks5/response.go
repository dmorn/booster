package socks5

type Response struct {
	Marshaler

	Ver      uint8
	Rep      uint8 // reponse code
	Rsv      uint8
	AddrType uint8
	BndAddr  []uint8
	BndPort  []uint8
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
	addrl := len(r.BndAddr)
	bas := 4         // BDN.ADDR start index
	bae := 4 + addrl // BDN.ADDR end index
	bufsize := 6 + addrl
	buf := make([]uint8, bufsize)

	buf[0] = r.Ver
	buf[1] = r.Rep
	buf[2] = FieldReserved
	buf[3] = r.AddrType
	copy(buf[bas:bas+addrl], r.BndAddr)
	copy(buf[bae:bae+2], r.BndPort)

	return buf, nil
}
