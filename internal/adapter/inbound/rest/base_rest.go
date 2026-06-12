package rest

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperror "notif-gateway/internal/value_object/handling/error"
)

// HandlerFunc is an HTTP handler that returns an error for RootHandler to map to JSON.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// RootHandler wraps HandlerFunc and writes apperror responses on failure.
func RootHandler(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			apperror.Write(w, r, err)
		}
	}
}

// RouteModule registers routes on a chi router under GetRoutePath() below HTTPAPIPrefix.
type RouteModule interface {
	GetRoutePath() string
	Mount(r chi.Router)
}

// Router exposes the full HTTP handler tree.
type Router interface {
	Router() http.Handler
}

// BaseRest is optional lifecycle for modules that own a sub-router.
type BaseRest interface {
	GetRoutePath() string
	Start(ctx context.Context) chi.Router
}
