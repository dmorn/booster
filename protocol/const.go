package protocol

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
