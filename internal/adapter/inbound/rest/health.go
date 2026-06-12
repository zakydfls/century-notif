package rest

import (
	"net/http"

	"notif-gateway/internal/value_object/dto/response"
)

func (h *Handler) health(w http.ResponseWriter, r *http.Request) error {
	response.SuccessData(w, "", http.StatusOK, map[string]string{"status": "ok"})
	return nil
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) error {
	response.SuccessData(w, "", http.StatusOK, map[string]string{"status": "ready"})
	return nil
}
