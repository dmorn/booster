/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package socks5

import (
	"context"
	"errors"
	"net"
)

// bind, not yet implemented. See RFC 1928
func (s *Socks5) Bind(ctx context.Context, conn net.Conn, target string) (net.Conn, error) {

	// cap is just an estimation
	buf := make([]byte, 0, 6+len(target))
	buf = append(buf, socks5Version, socks5RespCommandNotSupported, socks5FieldReserved)

	if _, err := conn.Write(buf); err != nil {
		return nil, errors.New("proxy: unable to write bind response: " + err.Error())
	}

	return nil, errors.New("proxy: bind command not supported")
}
