package rest

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

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
		startedAt := time.Now()
		wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		requestID := chimiddleware.GetReqID(r.Context())
		log := logger.WithRequestID(r.Context(), h.log, requestID)
		next.ServeHTTP(wrapped, r)

		status := wrapped.Status()
		if status == 0 {
			status = http.StatusOK
		}
		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int("bytes", wrapped.BytesWritten()),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		}
		switch {
		case status >= http.StatusInternalServerError:
			log.LogAttrs(r.Context(), slog.LevelError, "http request completed", attrs...)
		case status >= http.StatusBadRequest:
			log.LogAttrs(r.Context(), slog.LevelWarn, "http request completed", attrs...)
		default:
			log.LogAttrs(r.Context(), slog.LevelInfo, "http request completed", attrs...)
		}
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
