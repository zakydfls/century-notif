package response

import (
	errordto "notif-gateway/internal/value_object/dto/error"
	"notif-gateway/internal/value_object/dto/pagination"
)

const (
	StatusSuccess = "success"
	StatusError   = "error"

	DefaultSuccessMessage = "OK"
)

// Meta carries pagination totals for JSON responses without embedding full Pagination.Rows.
type Meta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalRows  int64 `json:"total_rows"`
	TotalPages int   `json:"total_pages"`
}

func MetaFromPagination(pg *pagination.Pagination) *Meta {
	if pg == nil {
		return nil
	}
	return &Meta{
		Page:       pg.GetPage(),
		Limit:      pg.GetLimit(),
		TotalRows:  pg.TotalRows,
		TotalPages: pg.TotalPages,
	}
}

type Response[T any] struct {
	Status    string            `json:"status"`
	RequestID string            `json:"request_id,omitempty"`
	Message   string            `json:"message,omitempty"` // English fallback
	Code      string            `json:"code,omitempty"`    // category, e.g. BAD_REQUEST
	ErrorKey  string            `json:"key,omitempty"`     // stable i18n key for FE
	Data      T                 `json:"data,omitempty"`
	Error     *errordto.Details `json:"-"`
	Meta      *Meta             `json:"meta,omitempty"`
}

type SuccessParams[T any] struct {
	RequestID string
	Message   string
	Data      T
	Meta      *Meta
}

func Success[T any](params SuccessParams[T]) Response[T] {
	message := params.Message
	if message == "" {
		message = DefaultSuccessMessage
	}
	return Response[T]{
		Status:    StatusSuccess,
		RequestID: params.RequestID,
		Message:   message,
		Data:      params.Data,
		Meta:      params.Meta,
	}
}

type FailParams struct {
	RequestID string
	Code      string
	ErrorKey  string
	Message   string
	Fields    map[string]errordto.FieldDetail
	Meta      *Meta
}

func Fail(params FailParams) Response[any] {
	resp := Response[any]{
		Status:    StatusError,
		RequestID: params.RequestID,
		Code:      params.Code,
		ErrorKey:  params.ErrorKey,
		Message:   params.Message,
		Meta:      params.Meta,
	}
	if len(params.Fields) > 0 {
		resp.Error = &errordto.Details{Fields: params.Fields}
	}
	return resp
}
