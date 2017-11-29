package socks5

import (
	"errors"
	"io"
	"net"
)

// Negotiate performs the very first method subnegotiation when handling
// a new connection. See RFC 1928.
func (s *Socks5) Negotiate(conn net.Conn) error {

	// len is just an estimation
	buf := make([]byte, 7)

	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return errors.New("proxy: failed to read negotiation: " + err.Error())
	}

	v := buf[0]         // protocol version
	nm := uint8(buf[1]) // number of methods

	if cap(buf) < int(nm) {
		buf = make([]byte, nm)
	} else {
		buf = buf[:nm]
	}

	// Check version number
	if v != socks5Version {
		return errors.New("proxy: unsupported version: " + string(v))
	}

	if _, err := io.ReadFull(conn, buf[:nm]); err != nil {
		return errors.New("proxy: failed to read methods: " + err.Error())
	}

	// select one method - could also be socksV5MethodNoAcceptableMethods
	m := acceptMethod(buf)

	buf = buf[:0]
	buf = append(buf, socks5Version)
	buf = append(buf, m)

	if _, err := conn.Write(buf); err != nil {
		return errors.New("proxy: unable to write negotitation reponse: " + err.Error())
	}

	return nil
}

func acceptMethod(m []uint8) uint8 {
	for _, sm := range supportedMethods {
		for _, tm := range m {
			if sm == tm {
				return sm
			}
		}
	}

	return socks5MethodNoAcceptableMethods
}
