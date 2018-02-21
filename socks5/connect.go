package socks5

import (
	"context"
	"errors"
	"net"
)

// connect dials a new connection with target, which must be a canonical
// address with host and port.
func (s *Socks5) Connect(ctx context.Context, conn net.Conn, target string) (net.Conn, error) {
	s.Printf("connect to %v (%v)", target, sha1Hash([]byte(target)))

	// cap is just an estimation
	buf := make([]byte, 0, 6+len(target))
	buf = append(buf, socks5Version)

	s.Lock()
	d := s.Dialer
	s.Unlock()

	tconn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		// TODO(daniel): Respond with proper code
		buf = append(buf, socks5RespHostUnreachable, socks5FieldReserved)
		if _, err := conn.Write(buf); err != nil {
			return nil, errors.New("socks5: unable to write connect response: " + err.Error())
		}

		return nil, err
	}
	// BUG: sometimes there is no err BUT the connection is nil
	if tconn == nil {
		return nil, errors.New("socks5: connect returned nil connection")
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

	return tconn, nil
}
