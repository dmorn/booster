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

package protocol

import (
	"fmt"
	"strings"
	"time"
)

type PayloadCtrl struct {
	Operation Operation
}

type PayloadBandwidth struct {
	Tot       int    `json:"tot"`
	Bandwidth int    `json:"bandwidth"`
	Type      string `json:"type"`
}

func (n *PayloadBandwidth) String() string {
	var b strings.Builder
	b.WriteString("net {\n")

	b.WriteString(fmt.Sprintf("\ttype: %s\n", n.Type))
	b.WriteString(fmt.Sprintf("\tbandwidth: %v, total: %v\n", n.Bandwidth, n.Tot))
	b.WriteString("\n}\n")

	return b.String()
}

type PayloadInspect struct {
	Features []Message
}

type PayloadHello struct {
	BPort string
	PPort string
}

type PayloadConnect struct {
	Target string
}

type PayloadDisconnect struct {
	ID string
}

type PayloadNode struct {
	ID      string    `json:"id"`
	BAddr   string    `json:"baddr"`
	PAddr   string    `json:"paddr"`
	Active  bool      `json:"active"`
	Tunnels []*Tunnel `json:"tunnels"`
}

func (n *PayloadNode) String() string {
	var b strings.Builder
	b.WriteString("node {\n")

	id := string([]byte(n.ID)[:10])
	b.WriteString(fmt.Sprintf("\tid: %v active: %v\n", id, n.Active))
	b.WriteString(fmt.Sprintf("\tba: %v pa: %v\n", n.BAddr, n.PAddr))
	b.WriteString("\ttunnels:\n")
	b.WriteString("\t[")
	if len(n.Tunnels) == 0 {
		b.WriteString("]")
	} else {
		b.WriteString("\n")
		for _, t := range n.Tunnels {
			b.WriteString(fmt.Sprintf("\t\t%v\n", t))
		}
		b.WriteString("\t]")
	}
	b.WriteString("\n}\n")

	return b.String()
}

type Tunnel struct {
	ID     string `json:"id"`
	Target string `json:"target"`
	ProxiedBy string `json:"proxied_by"`
	Acks   int    `json:"acks"`
	Copies int    `json:"copies"`
}

func (t *Tunnel) String() string {
	id := string([]byte(t.ID)[:10])
	return fmt.Sprintf(
		"{id: %v target: %v proxied_by: %v acks: %v copies %v}",
		id, t.Target, t.ProxiedBy, t.Acks, t.Copies,
	)
}

type PayloadHeartbeat struct {
	ID   string
	Hops int
	TTL  time.Time
}

type PayloadTunnelEvent struct {
	Target string
	Event  int
}
