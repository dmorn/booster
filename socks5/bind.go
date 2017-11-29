package socks5

import (
	"context"
	"errors"
	"net"
)

// Bind -- not yet implemented. See RFC 1928
func (s *Socks5) Bind(ctx context.Context, conn net.Conn, target string) (net.Conn, error) {

	// cap is just an estimation
	buf := make([]byte, 0, 6+len(target))
	buf = append(buf, socks5Version, socks5RespCommandNotSupported, socks5FieldReserved)

	if _, err := conn.Write(buf); err != nil {
		return nil, errors.New("proxy: unable to write bind response: " + err.Error())
	}

	return nil, errors.New("proxy: bind command not supported")
}
