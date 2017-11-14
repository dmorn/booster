package socks5

import (
	"context"
	"errors"
	"net"
)

// Connect dials a new connection with target, which must
// be a canonical address with host and port, according
// to RFC 1928.
func (s *Socks5) Connect(ctx context.Context, conn Conn, target string) (Conn, error) {

	// cap is just an estimation
	buf := make([]byte, 0, 6+len(target))
	buf = append(buf, socks5Version)

	tconn, err := s.getDialer().DialContext(ctx, "tcp", target)
	if err != nil {
		// TODO(daniel): Responde with proper code
		buf = append(buf, socks5RespHostUnreachable, socks5FieldReserved)
		if _, err := conn.Write(buf); err != nil {
			return nil, errors.New("proxy: unable to write connect response: " + err.Error())
		}

		return nil, err
	}

	buf = append(buf, socks5RespSuccess, socks5FieldReserved)

	// bnd addr
	addr := tconn.LocalAddr().(*net.TCPAddr)
	ip := addr.IP
	port := uint16(addr.Port)

	if ip4 := ip.To4(); ip4 != nil {
		buf = append(buf, socks5IP4)
		ip = ip4
	} else {
		buf = append(buf, socks5IP6)
	}
	buf = append(buf, ip...)

	// bdn port
	buf = append(buf, byte(port>>8), byte(port)&0xff)

	conn.Write(buf)

	s.Printf("CONNECT to %v. Response: %v\n", target, buf)

	return tconn, nil
}
