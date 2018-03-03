package protocol

const Version = "v0.1.0"

const (
	EncodingProtobuf uint8 = 1
)

const (
	ModuleHeader  string = "HE"
	ModulePayload        = "PA"
)

const (
	PacketOpeningTag  = ">"
	PacketClosingTag  = "<"
	PayloadOpeningTag = "["
	PayloadClosingTag = "]"
	Separator         = ":"
)

const (
	MessageHello int32 = 0
	MessageConnect = 1
	MessageNode = 2
)

// IsVersionSupported returns true if the current protocol version is compatible
// with the requested version.
func IsVersionSupported(v string) bool {
	// TODO(daniel): implement this check
	return v == Version
}
