package entity

import types "notif-gateway/internal/value_object/util/type"

type memo struct {
	value *types.Map[string, any] `json:"-" gorm:"-"`
}

func (b *memo) HasMemo(key string) bool {
	if b.value == nil {
		return false
	}
	_, ok := b.value.Get(key)
	return ok
}

func (b *memo) GetMemo(key string) any {
	if b.value == nil {
		return nil
	}
	return b.value.MustGet(key)
}

func (b *memo) GetMemoBool(key string) bool {
	if b.value == nil {
		return false
	}
	v, ok := b.value.Get(key)
	if !ok {
		return false
	}
	if val, ok := v.(bool); ok {
		return val
	}
	return false
}

func (b *memo) SetMemo(key string, value any) {
	if b.value == nil {
		b.value = types.NewMap[string, any]()
	}
	b.value.Set(key, value)
}

func (b *memo) RemoveMemo(key string) {
	if b.value == nil {
		return
	}
	b.value.Remove(key)
}

func (b *memo) ResetMemo() {
	b.value = nil
}
