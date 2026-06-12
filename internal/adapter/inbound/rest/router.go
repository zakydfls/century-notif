package rest

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"notif-gateway/internal/core/enum"
)

func (h *Handler) Router() http.Handler {
	router := chi.NewRouter()
	router.Use(requestIDMiddleware)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.Timeout(30 * time.Second))
	router.Use(h.requestLogger)

	router.Get("/healthz", RootHandler(h.health))
	router.Get("/readyz", RootHandler(h.ready))
	router.Route(enum.HTTPAPIPrefix, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(notificationKeyMiddleware(h.notificationAPIKey))
			r.Route(h.notifications.GetRoutePath(), h.notifications.Mount)
		})
		r.Route(h.webhooks.GetRoutePath(), h.webhooks.Mount)
	})
	return router
}
