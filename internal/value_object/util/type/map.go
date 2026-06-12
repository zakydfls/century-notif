package types

import (
	"maps"
)

type Map[K comparable, V any] struct {
	m map[K]V
}

func NewMap[K comparable, V any](initial ...map[K]V) *Map[K, V] {
	m := &Map[K, V]{m: make(map[K]V)}
	if len(initial) > 0 {
		maps.Copy(m.m, initial[0])
	}
	return m
}

func NewMapFromArray[K comparable, V any](arr []V, keyFunc func(v V) K) *Map[K, V] {
	m := NewMap[K, V]()
	for _, v := range arr {
		key := keyFunc(v)
		m.Set(key, v)
	}
	return m
}

func NewMapFromIArray[V any](arr []V) *Map[int, V] {
	m := NewMap[int, V]()
	for i, v := range arr {
		m.Set(i, v)
	}
	return m
}

func (m *Map[K, V]) Set(key K, value V) {
	m.m[key] = value
}

func (m *Map[K, V]) Has(key K) bool {
	_, ok := m.m[key]
	return ok
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	value, ok := m.m[key]
	return value, ok
}

func (m *Map[K, V]) MustGet(key K) V {
	value, ok := m.m[key]
	if !ok {
		return value
	}
	return value
}

func (m *Map[K, V]) Remove(key K) {
	delete(m.m, key)
}

func (m *Map[K, V]) Keys() []K {
	keys := make([]K, 0, len(m.m))
	for key := range m.m {
		keys = append(keys, key)
	}
	return keys
}

func (m *Map[K, V]) FindMissingKeys(lookupKeys ...K) []K {
	missingKeys := NewSet[K]()
	for _, candidateKey := range lookupKeys {
		if _, exists := m.m[candidateKey]; !exists {
			missingKeys.Add(candidateKey)
		}
	}
	return missingKeys.Values()
}

func (m *Map[K, V]) Values() []V {
	values := make([]V, 0, len(m.m))
	for _, value := range m.m {
		values = append(values, value)
	}
	return values
}

func (m *Map[K, V]) ForEach(fn func(key K, value V)) {
	for key, value := range m.m {
		fn(key, value)
	}
}

func (m *Map[K, V]) Size() int {
	return len(m.m)
}

func (m *Map[K, V]) IsEmpty() bool {
	return m.Size() == 0
}

func (m *Map[K, V]) Clear() {
	m.m = make(map[K]V)
}
