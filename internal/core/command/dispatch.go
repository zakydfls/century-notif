package command

import (
	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type Dispatch struct {
	NotificationID string
	Channel        enum.Channel
	Recipient      string
	Body           string

	ProviderTemplateID   string
	ProviderTemplateVars types.JSONMap
	Metadata             types.JSONMap
}
