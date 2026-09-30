package apperror

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	responsedto "notif-gateway/internal/value_object/dto/response"
)

type Mapper func(err error) *Error

var (
	mappersMu sync.RWMutex
	mappers   []Mapper
)

func RegisterMapper(fn Mapper) {
	if fn == nil {
		return
	}
	mappersMu.Lock()
	defer mappersMu.Unlock()
	mappers = append(mappers, fn)
}

func Resolve(err error) *Error {
	if err == nil {
		return ErrInternal(nil)
	}
	var ae *Error
	if errors.As(err, &ae) {
		return ae
	}
	mappersMu.RLock()
	defer mappersMu.RUnlock()
	for _, m := range mappers {
		if e := m(err); e != nil {
			return e
		}
	}
	return ErrInternal(err)
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	e := Resolve(err)
	code, key, message, fields := FailParams(e)
	if r != nil {
		attrs := []slog.Attr{
			slog.String("request_id", chimiddleware.GetReqID(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", e.HTTPStatus),
			slog.String("code", code),
			slog.String("error_key", key),
			slog.String("error", e.Error()),
		}
		if e.HTTPStatus >= http.StatusInternalServerError {
			slog.LogAttrs(r.Context(), slog.LevelError, "http request failed", attrs...)
		} else {
			slog.LogAttrs(r.Context(), slog.LevelWarn, "http request failed", attrs...)
		}
		responsedto.FailFromCtx(r.Context(), w, e.HTTPStatus, code, key, message, fields)
		return
	}
	responsedto.FailData(w, e.HTTPStatus, code, key, message, fields)
}
