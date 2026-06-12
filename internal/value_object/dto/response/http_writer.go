package response

import (
	"context"
	"encoding/json"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"notif-gateway/internal/core/enum"
	errordto "notif-gateway/internal/value_object/dto/error"
)

// RequestID returns the request id from ctx (e.g. chi RequestID middleware), or a new UUID if missing.
func RequestID(ctx context.Context) string {
	if ctx != nil {
		if id := chimiddleware.GetReqID(ctx); id != "" {
			return id
		}
	}
	return uuid.New().String()
}

func ensureRequestIDHeader(w http.ResponseWriter, id string) {
	if w.Header().Get(enum.HeaderRequestID) == "" {
		w.Header().Set(enum.HeaderRequestID, id)
	}
}

// SuccessData writes a JSON success envelope. request_id is a new UUID (short call; no request/context).
// For the same id as chi RequestID middleware, use SuccessDataFromCtx.
func SuccessData(w http.ResponseWriter, message string, status int, data any) {
	id := uuid.New().String()
	ensureRequestIDHeader(w, id)
	w.Header().Set(enum.HeaderContentType, enum.HeaderApplicationJSON)
	w.WriteHeader(status)
	msg := message
	if msg == "" {
		msg = DefaultSuccessMessage
	}
	_ = json.NewEncoder(w).Encode(Success(SuccessParams[any]{
		RequestID: id,
		Message:   msg,
		Data:      data,
		Meta:      nil,
	}))
}

// SuccessDataFromCtx is like SuccessData but uses RequestID(ctx) so body/header match chi middleware when present.
func SuccessDataFromCtx(ctx context.Context, w http.ResponseWriter, message string, status int, data any) {
	id := RequestID(ctx)
	ensureRequestIDHeader(w, id)
	w.Header().Set(enum.HeaderContentType, enum.HeaderApplicationJSON)
	w.WriteHeader(status)
	msg := message
	if msg == "" {
		msg = DefaultSuccessMessage
	}
	_ = json.NewEncoder(w).Encode(Success(SuccessParams[any]{
		RequestID: id,
		Message:   msg,
		Data:      data,
		Meta:      nil,
	}))
}

// SuccessPaginate writes a JSON success envelope with data and pagination meta (new UUID per response; see SuccessPaginateFromCtx).
func SuccessPaginate(w http.ResponseWriter, message string, status int, data any, meta *Meta) {
	id := uuid.New().String()
	ensureRequestIDHeader(w, id)
	w.Header().Set(enum.HeaderContentType, enum.HeaderApplicationJSON)
	w.WriteHeader(status)
	msg := message
	if msg == "" {
		msg = DefaultSuccessMessage
	}
	_ = json.NewEncoder(w).Encode(Success(SuccessParams[any]{
		RequestID: id,
		Message:   msg,
		Data:      data,
		Meta:      meta,
	}))
}

// SuccessPaginateFromCtx is like SuccessPaginate but uses RequestID(ctx) for middleware alignment.
func SuccessPaginateFromCtx(ctx context.Context, w http.ResponseWriter, message string, status int, data any, meta *Meta) {
	id := RequestID(ctx)
	ensureRequestIDHeader(w, id)
	w.Header().Set(enum.HeaderContentType, enum.HeaderApplicationJSON)
	w.WriteHeader(status)
	msg := message
	if msg == "" {
		msg = DefaultSuccessMessage
	}
	_ = json.NewEncoder(w).Encode(Success(SuccessParams[any]{
		RequestID: id,
		Message:   msg,
		Data:      data,
		Meta:      meta,
	}))
}

// FailData writes the JSON error envelope (new request_id; see FailFromCtx for chi-aligned id).
func FailData(w http.ResponseWriter, httpStatus int, code, errorKey, message string, fields map[string]errordto.FieldDetail) {
	id := uuid.New().String()
	ensureRequestIDHeader(w, id)
	w.Header().Set(enum.HeaderContentType, enum.HeaderApplicationJSON)
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(Fail(FailParams{
		RequestID: id,
		Code:      code,
		ErrorKey:  errorKey,
		Message:   message,
		Fields:    fields,
	}))
}

// FailFromCtx is like FailData but uses RequestID(ctx) so body/header match chi middleware when present.
func FailFromCtx(ctx context.Context, w http.ResponseWriter, httpStatus int, code, errorKey, message string, fields map[string]errordto.FieldDetail) {
	id := RequestID(ctx)
	ensureRequestIDHeader(w, id)
	w.Header().Set(enum.HeaderContentType, enum.HeaderApplicationJSON)
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(Fail(FailParams{
		RequestID: id,
		Code:      code,
		ErrorKey:  errorKey,
		Message:   message,
		Fields:    fields,
	}))
}
