package debugger

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/k0kubun/pp/v3"
)

var ErrCauseDD = errors.New("error caused by dd")

func PrettyPrint(vs ...any) {
	slog.Debug("🚨🚨🚨🚨🚨 PrettyPrint 🚨🚨🚨🚨🚨🚨🚨")
	pp.Println(vs...)
	slog.Debug("🚨🚨🚨🚨🚨 PrettyPrint 🚨🚨🚨🚨🚨🚨🚨")
}

func DiePrettyPrint(vs ...any) {
	PrettyPrint(vs...)
	panic(ErrCauseDD)
}

func DiePrint(vs ...any) {
	fmt.Println(vs...)
	panic(ErrCauseDD)
}

func PrettyJSON(vs ...any) {
	fmt.Println("🚨🚨🚨🚨🚨 PrettyJSON 🚨🚨🚨🚨🚨🚨🚨")
	for _, v := range vs {
		pretty := prepareForJSON(v)
		jsonData, err := json.MarshalIndent(pretty, "", "  ")
		if err != nil {
			fmt.Printf("JSON encode error: %v\n", err)
		} else {
			fmt.Println(string(jsonData))
		}
	}
	fmt.Println("🚨🚨🚨🚨🚨 PrettyJSON 🚨🚨🚨🚨🚨🚨🚨")
}

func DiePrettyJSON(vs ...any) {
	PrettyJSON(vs...)
	panic(ErrCauseDD)
}

func prepareForJSON(v any) any {
	val := reflect.ValueOf(v)

	if !val.IsValid() || (val.Kind() == reflect.Ptr && val.IsNil()) {
		return nil
	}

	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	return val.Interface()
}
