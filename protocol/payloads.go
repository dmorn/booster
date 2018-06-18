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
	"time"
)

type PayloadNetworkUpdate struct {
	NodeID     string       `json:"node_id"`
	Operation  Operation    `json:"operation"`
	RemoteNode *PayloadNode `json:"remote_node"`
}

type PayloadProxyUpdate struct {
	NodeID    string    `json:"node_id"`
	Target    string    `json:"target"`
	Operation Operation `json:"operation"`
}

type PayloadCtrl struct {
	Operation Operation
}

type PayloadBandwidth struct {
	NodeID    string `json:"-"`
	Tot       int    `json:"tot"`
	Bandwidth int    `json:"bandwidth"`
	Type      string `json:"type"`
}

type PayloadMonitor struct {
	Feature MonitorFeature
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

type Tunnel struct {
	ID     string `json:"-"`
	Target string `json:"target"`
	Copies int    `json:"copies"`
}

type PayloadHeartbeat struct {
	ID   string
	Hops int
	TTL  time.Time
}
