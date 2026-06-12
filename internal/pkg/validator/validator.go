package validator

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	playground "github.com/go-playground/validator/v10"

	errordto "notif-gateway/internal/value_object/dto/error"
	apperror "notif-gateway/internal/value_object/handling/error"
)

var (
	once     sync.Once
	instance *playground.Validate
)

func engine() *playground.Validate {
	once.Do(func() {
		instance = playground.New(playground.WithRequiredStructEnabled())
		instance.RegisterTagNameFunc(func(field reflect.StructField) string {
			name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name == "" {
				return field.Name
			}
			return name
		})
	})
	return instance
}

// Bind decodes JSON body into target and validates it.
// Returns *apperror.Error: ErrBadRequest on decode errors, ErrUnprocessableEntity on validation errors.
func Bind(r *http.Request, target any) error {
	if r == nil || r.Body == nil {
		return apperror.ErrBadRequest(nil, "request body is required", apperror.KeyRequestBodyRequired)
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return apperror.ErrBadRequest(err, "request body is required", apperror.KeyRequestBodyRequired)
		}
		return apperror.ErrBadRequest(err, "invalid json body: "+err.Error(), apperror.KeyInvalidJSONBody)
	}
	return Validate(target)
}

// BindQuery decodes URL query into target and validates it.
func BindQuery(r *http.Request, target any) error {
	if r == nil {
		return apperror.ErrBadRequest(nil, "request is required", apperror.KeyRequestRequired)
	}
	if err := decodeQuery(r.URL.Query(), target); err != nil {
		return apperror.ErrBadRequest(err, "invalid query parameter: "+err.Error(), apperror.KeyInvalidQueryParam)
	}
	return Validate(target)
}

func decodeQuery(values map[string][]string, target any) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("target must be a non-nil pointer to a struct")
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return errors.New("target must point to a struct")
	}
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		key := tagName(field, "query")
		if key == "" {
			key = tagName(field, "schema")
		}
		if key == "" {
			key = strings.ToLower(field.Name)
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Struct && field.Anonymous {
			if err := decodeQuery(values, fv.Addr().Interface()); err != nil {
				return err
			}
			continue
		}
		raw, ok := values[key]
		if !ok || len(raw) == 0 || raw[0] == "" {
			continue
		}
		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	return nil
}

func tagName(field reflect.StructField, tag string) string {
	value := field.Tag.Get(tag)
	if value == "" || value == "-" {
		return ""
	}
	return strings.SplitN(value, ",", 2)[0]
}

func setField(fv reflect.Value, raw []string) error {
	if !fv.CanSet() {
		return errors.New("unsettable field")
	}
	if fv.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(fv.Type(), len(raw), len(raw))
		for i, item := range raw {
			if err := setScalar(slice.Index(i), item); err != nil {
				return err
			}
		}
		fv.Set(slice)
		return nil
	}
	return setScalar(fv, raw[0])
}

func setScalar(fv reflect.Value, raw string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(f)
	default:
		return fmt.Errorf("unsupported field kind %s", fv.Kind())
	}
	return nil
}

// Validate runs struct-tag validation. Returns *apperror.Error (ErrUnprocessableEntity) on failure.
func Validate(target any) error {
	if err := engine().Struct(target); err != nil {
		return toAppError(err)
	}
	return nil
}

func toAppError(err error) *apperror.Error {
	var verrs playground.ValidationErrors
	if !errors.As(err, &verrs) {
		return apperror.ErrUnprocessableEntity(err, err.Error(), nil, apperror.KeyValidationFailed)
	}
	fields := make(map[string]errordto.FieldDetail, len(verrs))
	messages := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		field := fe.Field()
		message := humanMessage(field, fe.Tag(), fe.Param())
		detail := errordto.FieldDetail{
			Key:     validationFieldKey(field, fe.Tag()),
			Rule:    fe.Tag(),
			Message: message,
		}
		if fe.Param() != "" {
			detail.Param = fe.Param()
		}
		fields[field] = detail
		messages = append(messages, message)
	}
	return apperror.ErrUnprocessableEntity(err, strings.Join(messages, "; "), fields, apperror.KeyValidationFailed)
}

func validationFieldKey(field, rule string) string {
	return fmt.Sprintf("errors.validation.%s.%s", strings.ToLower(field), rule)
}

func humanMessage(field, rule, param string) string {
	switch rule {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, param)
	case "len":
		return fmt.Sprintf("%s must have length %s", field, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email", field)
	case "uuid", "uuid4":
		return fmt.Sprintf("%s must be a valid UUID", field)
	case "e164":
		return fmt.Sprintf("%s must be a valid E.164 phone number", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "dive":
		return fmt.Sprintf("%s contains invalid item", field)
	default:
		if param != "" {
			return fmt.Sprintf("%s failed %q rule (%s)", field, rule, param)
		}
		return fmt.Sprintf("%s failed %q rule", field, rule)
	}
}
