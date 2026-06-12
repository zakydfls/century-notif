package template

import (
	"notif-gateway/internal/core/enum"
	types "notif-gateway/internal/value_object/util/type"
)

type RenderCommand struct {
	Key      string
	Channel  enum.Channel
	Provider enum.Provider
	Params   types.JSONMap
}

type RenderResult struct {
	Body               string
	ProviderTemplateID string
}
