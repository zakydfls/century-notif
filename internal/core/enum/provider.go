package enum

type Provider string

const (
	ProviderTwilio    Provider = "twilio"
	ProviderAlibaba   Provider = "alibaba"
	ProviderWAHA      Provider = "waha"
	ProviderWhatsockt Provider = "whatsockt"
	ProviderGoDaddy   Provider = "godaddy"
	ProviderResend    Provider = "resend"
)

func (p Provider) String() string {
	return string(p)
}

func (p Provider) IsValid() bool {
	switch p {
	case ProviderTwilio, ProviderAlibaba, ProviderWAHA, ProviderWhatsockt, ProviderGoDaddy, ProviderResend:
		return true
	default:
		return false
	}
}

func (p Provider) Supports(channel Channel) bool {
	switch channel {
	case ChannelSMS:
		return p == ProviderTwilio || p == ProviderAlibaba
	case ChannelWhatsApp:
		return p == ProviderTwilio || p == ProviderWAHA || p == ProviderWhatsockt
	case ChannelEmail:
		return p == ProviderGoDaddy || p == ProviderResend
	default:
		return false
	}
}
