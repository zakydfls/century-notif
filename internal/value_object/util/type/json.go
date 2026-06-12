package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type JSONMap map[string]any

func NewJSONMap(value map[string]any) JSONMap {
	if value == nil {
		return JSONMap{}
	}
	return JSONMap(value)
}

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]any(m))
}

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return errors.New("unsupported json map scan type")
	}
	if len(raw) == 0 {
		*m = JSONMap{}
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*m = JSONMap(decoded)
	return nil
}
