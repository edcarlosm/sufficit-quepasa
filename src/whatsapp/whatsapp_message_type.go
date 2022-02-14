package whatsapp

type WhatsappMessageType uint

const (
	UnknownMessageType WhatsappMessageType = iota
	ImageMessageType
	DocumentMessageType
	AudioMessageType
	VideoMessageType
	TextMessageType
)

func (Type WhatsappMessageType) String() string {
	switch Type {
	case ImageMessageType:
		return "image"
	case DocumentMessageType:
		return "document"
	case AudioMessageType:
		return "audio"
	case VideoMessageType:
		return "video"
	case TextMessageType:
		return "text"
	}

	return "unknown"
}
