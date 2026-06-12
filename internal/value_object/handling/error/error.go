package apperror

import (
	"errors"
	"net/http"

	errordto "notif-gateway/internal/value_object/dto/error"
)

// Response code categories (monitoring / logging). FE translates via Error.Key (error_key in JSON).
const (
	CodeInternal       = "INTERNAL_ERROR"
	CodeBadRequest     = "BAD_REQUEST"
	CodeValidation     = "VALIDATION_ERROR"
	CodeNotFound       = "NOT_FOUND"
	CodeConflict       = "CONFLICT"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeProviderFailed = "PROVIDER_FAILED"
	CodeNotImplemented = "NOT_IMPLEMENTED"
)

// ClientError is what non-rest layers can return without importing net/http or the response package.
type ClientError interface {
	error
	GetStatus() int
	GetCode() string
	GetKey() string
	GetMessage() string
}

type Error struct {
	Code       string
	Key        string // i18n key for FE (JSON: error_key)
	Message    string // English fallback / ops detail
	HTTPStatus int
	Fields     map[string]errordto.FieldDetail
	Cause      error
}

func (e *Error) Error() string {
	if e.Message != "" {
		if e.Cause == nil {
			return e.Message
		}
		return e.Message + ": " + e.Cause.Error()
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	if e.Key != "" {
		return e.Key
	}
	return e.Code
}

func (e *Error) Unwrap() error      { return e.Cause }
func (e *Error) GetStatus() int     { return e.HTTPStatus }
func (e *Error) GetCode() string    { return e.Code }
func (e *Error) GetKey() string     { return e.Key }
func (e *Error) GetMessage() string { return e.Message }

type NewParams struct {
	Code       string
	Key        string
	Message    string
	HTTPStatus int
	Fields     map[string]errordto.FieldDetail
	Cause      error
}

func New(params NewParams) *Error {
	status := params.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	key := params.Key
	if key == "" {
		key = params.Code
	}
	message := params.Message
	if message == "" && params.Cause != nil {
		message = params.Cause.Error()
	}
	if message == "" {
		message = http.StatusText(status)
	}
	return &Error{
		Code:       params.Code,
		Key:        key,
		Message:    message,
		HTTPStatus: status,
		Fields:     params.Fields,
		Cause:      params.Cause,
	}
}

func optionalKey(key ...string) string {
	if len(key) == 0 {
		return ""
	}
	return key[0]
}

// ErrBadRequest — cause and message are required; key is optional (defaults to CodeBadRequest).
func ErrBadRequest(cause error, message string, key ...string) *Error {
	return New(NewParams{
		Code: CodeBadRequest, Key: optionalKey(key...), Message: message,
		HTTPStatus: http.StatusBadRequest, Cause: cause,
	})
}

// ErrBadRequestCode — code is required when not BAD_REQUEST; key is optional.
func ErrBadRequestCode(cause error, message, code string, key ...string) *Error {
	return New(NewParams{
		Code: code, Key: optionalKey(key...), Message: message,
		HTTPStatus: http.StatusBadRequest, Cause: cause,
	})
}

func ErrUnauthorized(cause error, message string, key ...string) *Error {
	if message == "" {
		message = "unauthorized"
	}
	k := optionalKey(key...)
	if k == "" {
		k = KeyUnauthorized
	}
	return New(NewParams{
		Code: CodeUnauthorized, Key: k, Message: message,
		HTTPStatus: http.StatusUnauthorized, Cause: cause,
	})
}

func ErrForbidden(cause error, message string, key ...string) *Error {
	if message == "" {
		message = "forbidden"
	}
	k := optionalKey(key...)
	if k == "" {
		k = KeyForbidden
	}
	return New(NewParams{
		Code: CodeForbidden, Key: k, Message: message,
		HTTPStatus: http.StatusForbidden, Cause: cause,
	})
}

func ErrNotFound(cause error, message string, key ...string) *Error {
	if message == "" {
		message = "resource not found"
	}
	k := optionalKey(key...)
	if k == "" {
		k = KeyRecordNotFound
	}
	return New(NewParams{
		Code: CodeNotFound, Key: k, Message: message,
		HTTPStatus: http.StatusNotFound, Cause: cause,
	})
}

func ErrConflict(cause error, message string, key ...string) *Error {
	return New(NewParams{
		Code: CodeConflict, Key: optionalKey(key...), Message: message,
		HTTPStatus: http.StatusConflict, Cause: cause,
	})
}

// ErrUnprocessableEntity — fields may be nil; key is optional (defaults to KeyValidationFailed).
func ErrUnprocessableEntity(cause error, message string, fields map[string]errordto.FieldDetail, key ...string) *Error {
	k := optionalKey(key...)
	if k == "" {
		k = KeyValidationFailed
	}
	return New(NewParams{
		Code: CodeValidation, Key: k, Message: message,
		HTTPStatus: http.StatusUnprocessableEntity, Fields: fields, Cause: cause,
	})
}

func ErrInternal(cause error) *Error {
	return New(NewParams{
		Code: CodeInternal, Key: KeyInternal, Message: "internal server error",
		HTTPStatus: http.StatusInternalServerError, Cause: cause,
	})
}

func ErrNotImplemented(cause error, message string, key ...string) *Error {
	if message == "" {
		message = "not implemented"
	}
	k := optionalKey(key...)
	if k == "" {
		k = KeyNotImplemented
	}
	return New(NewParams{
		Code: CodeNotImplemented, Key: k, Message: message,
		HTTPStatus: http.StatusNotImplemented, Cause: cause,
	})
}

func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal(err)
}

func FailParams(err *Error) (code, key, message string, fields map[string]errordto.FieldDetail) {
	if err == nil {
		err = ErrInternal(nil)
	}
	return err.Code, err.Key, err.Message, err.Fields
}
