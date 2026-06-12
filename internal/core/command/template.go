package command

import (
	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type RenderTemplate struct {
	Key      string
	Channel  enum.Channel
	Provider enum.Provider
	Params   types.JSONMap
}

type RenderTemplateResult struct {
	Body               string
	ProviderTemplateID string
}
