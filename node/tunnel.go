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

package node

import (
	"sync"
)

type Tunnel struct {
	id     string
	Target string

	sync.Mutex
	copies int // number of copies
}

func NewTunnel(target string) *Tunnel {
	return &Tunnel{
		id:     sha1Hash([]byte(target)),
		Target: target,
		copies: 1,
	}
}

func (t *Tunnel) ID() string {
	return t.id
}

func (t *Tunnel) Copies() int {
	t.Lock()
	defer t.Unlock()

	return t.copies
}

func (t *Tunnel) Copy() *Tunnel {
	return &Tunnel{
		id:     t.ID(),
		Target: t.Target,
		copies: t.Copies(),
	}
}
