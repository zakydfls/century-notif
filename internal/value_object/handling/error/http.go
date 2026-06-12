package apperror

import (
	"errors"
	"net/http"
	"sync"

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
		responsedto.FailFromCtx(r.Context(), w, e.HTTPStatus, code, key, message, fields)
		return
	}
	responsedto.FailData(w, e.HTTPStatus, code, key, message, fields)
}
