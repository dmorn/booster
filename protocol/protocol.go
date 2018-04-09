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

// Package protocol provides functionalities to create payloads and compose packet
// that conform to the booster protocol.
package protocol

// Booster protocol version
const Version = "v0.1.0"

// Possible encodings
const (
	EncodingProtobuf uint8 = 1
)

// Module Identifiers
const (
	ModuleHeader  string = "HE"
	ModulePayload        = "PA"
)

// Tags used in the encoding and decoding of packets.
const (
	PacketOpeningTag  = ">"
	PacketClosingTag  = "<"
	PayloadOpeningTag = "["
	PayloadClosingTag = "]"
	Separator         = ":"
)

type Message int32

// Booster possible packet messages
const (
	MessageHello      Message = 0
	MessageConnect            = 1
	MessageDisconnect         = 2
	MessageNode               = 3
	MessageHeartbeat          = 4
	MessageTunnel             = 5
	MessageNotify             = 6
	MessageInspect            = 7
	MessageBandwidth          = 8
)

// Tunnel operations
const (
	TunnelAck    int32 = 0
	TunnelRemove       = 1
)

// IsVersionSupported returns true if the current protocol version is compatible
// with the requested version.
func IsVersionSupported(v string) bool {
	// TODO(daniel): implement this check
	return v == Version
}
