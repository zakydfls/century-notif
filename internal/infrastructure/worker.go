package infrastructure

import "notif-gateway/internal/core/enum"

type ChannelWorkerConfig struct {
	Prefetch    int
	Concurrency int
	MaxRetry    int
}

type ChannelWorkers struct {
	Default  ChannelWorkerConfig
	Channels map[enum.Channel]ChannelWorkerConfig
}

// NewChannelWorkers builds worker settings for the MQ consumer. Env supplies defaults; edit Channels for per-queue tuning.
func NewChannelWorkers(defaults ChannelWorkerConfig) ChannelWorkers {
	return ChannelWorkers{
		Default: defaults,
		Channels: map[enum.Channel]ChannelWorkerConfig{
			enum.ChannelEmail:    {Concurrency: 5, Prefetch: 5},
			enum.ChannelSMS:      {Concurrency: 10, Prefetch: 10},
			enum.ChannelWhatsApp: {Concurrency: 6, Prefetch: 6},
		},
	}
}

func (w ChannelWorkers) ForChannel(ch enum.Channel) ChannelWorkerConfig {
	base := w.Default
	override, ok := w.Channels[ch]
	if !ok {
		return base
	}
	if override.Prefetch > 0 {
		base.Prefetch = override.Prefetch
	}
	if override.Concurrency > 0 {
		base.Concurrency = override.Concurrency
	}
	if override.MaxRetry > 0 {
		base.MaxRetry = override.MaxRetry
	}
	return base
}
