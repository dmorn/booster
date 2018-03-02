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
