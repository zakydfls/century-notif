package enum

type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
	ChannelEmail    Channel = "email"
)

func (c Channel) String() string {
	return string(c)
}

func (c Channel) IsValid() bool {
	switch c {
	case ChannelSMS, ChannelWhatsApp, ChannelEmail:
		return true
	default:
		return false
	}
}
