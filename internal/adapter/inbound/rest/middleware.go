package rest

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	"notif-gateway/internal/pkg/logger"
	apperror "notif-gateway/internal/value_object/handling/error"
)

const unknownServiceName = "unknown"

type serviceNameContextKey struct{}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(enum.HeaderRequestID))
		if parsed, err := uuid.Parse(id); err != nil {
			id = uuid.New().String()
		} else {
			id = parsed.String()
		}
		ctx := context.WithValue(r.Context(), chimiddleware.RequestIDKey, id)
		w.Header().Set(enum.HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := chimiddleware.GetReqID(r.Context())
		log := logger.WithRequestID(r.Context(), h.log, requestID)
		log.InfoContext(r.Context(), "http request", slog.String("method", r.Method), slog.String("path", r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func notificationKeyMiddleware(expectedKey string) func(http.Handler) http.Handler {
	expected := []byte(expectedKey)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := strings.TrimSpace(r.Header.Get(enum.HeaderNotificationKey))
			if len(got) != len(expected) {
				apperror.Write(w, r, apperror.ErrUnauthorized(nil, "", apperror.KeyUnauthorized))
				return
			}
			if subtle.ConstantTimeCompare([]byte(got), expected) != 1 {
				apperror.Write(w, r, apperror.ErrUnauthorized(nil, "", apperror.KeyUnauthorized))
				return
			}
			ctx := context.WithValue(r.Context(), serviceNameContextKey{}, serviceNameFromHeader(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func serviceNameFromHeader(r *http.Request) string {
	if r == nil {
		return unknownServiceName
	}
	serviceName := strings.TrimSpace(r.Header.Get(enum.HeaderCenturyService))
	if serviceName == "" {
		return unknownServiceName
	}
	return serviceName
}

func serviceNameFromContext(ctx context.Context) string {
	serviceName, _ := ctx.Value(serviceNameContextKey{}).(string)
	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return unknownServiceName
	}
	return serviceName
}
