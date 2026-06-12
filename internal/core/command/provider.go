package command

import (
	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type ProviderSend struct {
	NotificationID string
	Channel        enum.Channel
	Provider       enum.Provider
	Recipient      string
	Body           string

	ProviderTemplateID   string
	ProviderTemplateVars types.JSONMap
	Metadata             types.JSONMap
}

type ProviderSendResult struct {
	MessageID string
	Raw       types.JSONMap
}
