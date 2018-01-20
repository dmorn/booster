package socks5

import (
	"context"
	"errors"
	"net"
)

// associate -- not yet implemented. See RFC 1928
func (s *Socks5) Associate(ctx context.Context, conn net.Conn, target string) (net.Conn, error) {

	// cap is just an estimation
	buf := make([]byte, 0, 6+len(target))
	buf = append(buf, socks5Version, socks5RespCommandNotSupported, socks5FieldReserved)

	if _, err := conn.Write(buf); err != nil {
		return nil, errors.New("proxy: unable to write associate response: " + err.Error())
	}

	return nil, errors.New("proxy: associate command not supported")
}
