package types

import (
	"slices"

	"github.com/google/uuid"
)

// Set implements a generic set data structure that guarantees uniqueness of elements
//
// Use when you need to maintain a collection of unique values with fast lookup operations
type Set[T comparable] struct {
	m        *Map[T, struct{}]
	arr      []T
	readonly bool
}

// ReadOnly creates a read-only view of the set
//
// # Use when you want to prevent modifications to the set
//
// Example: readOnlySet := ReadOnly(mySet) // readOnlySet cannot be modified
func ReadOnly[T comparable](s *Set[T]) *Set[T] {
	return s.readOnly()
}

// newSet creates a new set initialized with the provided values
//
// # Use when you want to create a set with specific initial values
//
// Example: mySet := newSet(1, 2, 3) // Creates a set with elements 1, 2, and 3
//
// Note: This function is not exported, use NewSet for public access
func newSet[T comparable](initial ...T) *Set[T] {
	s := &Set[T]{
		m:        NewMap[T, struct{}](),
		arr:      make([]T, 0),
		readonly: false,
	}
	if len(initial) > 0 {
		s.Add(initial...)
	}
	return s
}

// NewSet creates a new empty set for any comparable type T
//
// # Use when you need to instantiate a set for a specific type
//
// Example: intSet := NewSet[int]() // Creates an empty set for integers
func NewSet[T comparable](initial ...T) *Set[T] {
	return newSet(initial...)
}

// NewSetUUID creates a specialized empty set for UUID values
//
// # Use when working with collections of unique identifiers
//
// Example: idSet := NewSetUUID(); idSet.Add(uuid.New())
func NewSetUUID(initial ...uuid.UUID) *Set[uuid.UUID] {
	return newSet(initial...)
}

// NewSetString creates a specialized empty set for string values
//
// # Use when you need to track unique string values like codes or names
//
// Example: nameSet := NewSetString(); nameSet.Add("john", "alice")
func NewSetString(initial ...string) *Set[string] {
	return newSet(initial...)
}

func NewSetFromArray[K comparable, V any](arr []V, keyFunc func(v V) K) *Set[K] {
	s := newSet[K]()
	if len(arr) == 0 {
		return s
	}
	for _, v := range arr {
		k := keyFunc(v)
		s.Add(k)
	}
	return s
}

func (s *Set[T]) readOnly() *Set[T] {
	return &Set[T]{m: s.m, readonly: true}
}

func (s *Set[T]) IsReadOnly() bool {
	return s.readonly
}

// Add inserts one or more values into the set, maintaining uniqueness
//
// # Use when you need to add elements without creating duplicates
//
// Example: mySet.Add(1, 2, 3) // Adds elements 1, 2, and 3 to the set
func (s *Set[T]) Add(values ...T) {
	if len(values) == 0 {
		return
	}
	if s.IsReadOnly() {
		return
	}
	for _, value := range values {
		if s.Contains(value) {
			return
		}
		s.m.Set(value, struct{}{})
	}
}

// Remove deletes one or more values from the set if they exist
//
// # Use when you need to eliminate specific elements from your collection
//
// Example: mySet.Remove(1, 2) // Removes elements 1 and 2 from the set
func (s *Set[T]) Remove(values ...T) {
	if len(values) == 0 {
		return
	}
	if s.IsReadOnly() {
		return
	}
	for _, value := range values {
		if !s.Contains(value) {
			return
		}
		s.m.Remove(value)
	}
}

// Union combines elements from another set into this set
//
// # Use when you need to merge two sets while maintaining uniqueness
//
// Example: set1.Union(set2) // All elements from set2 are added to set1
func (s *Set[T]) Union(other *Set[T]) {
	if other == nil {
		return
	}
	if s.IsReadOnly() {
		return
	}
	s.m.ForEach(func(key T, value struct{}) {
		s.Add(key)
	})
}

// Intersect keeps only elements that exist in both this set and another set
//
// # Use when you need to find common elements between two collections
//
// Example: set1.Intersect(set2) // set1 will only contain elements present in both sets
func (s *Set[T]) Intersect(other *Set[T]) {
	if other == nil {
		return
	}
	if s.IsReadOnly() {
		return
	}
	s.m.ForEach(func(key T, value struct{}) {
		if !other.Contains(key) {
			s.Remove(key)
		}
	})
}

// Index returns the index of a specific value in the set
//
// # Use when you need to find the position of an element in the set
//
// Example: index := mySet.Index(2) // Returns the index of element 2 in the set
func (s *Set[T]) Index(value T) int {
	if len(s.arr) != s.Size() {
		s.Values()
	}
	index := slices.Index(s.arr, value)
	return index
}

// Contains checks if a specific value exists in the set
//
// # Use for fast O(1) lookups to determine set membership
//
// Example: if mySet.Contains(5) { fmt.Println("5 is in the set") }
func (s *Set[T]) Contains(value T) bool {
	_, ok := s.m.Get(value)
	return ok
}

// Difference returns a new set containing elements in this set that are not in the other set
//
// # Use when you need to find elements unique to one set compared to another
//
// Example: diffSet := set1.Difference(set2) // diffSet contains elements in set1 but not in set2
func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	if other == nil {
		return s
	}
	diff := NewSet[T]()
	s.m.ForEach(func(key T, value struct{}) {
		if !other.Contains(key) {
			diff.Add(key)
		}
	})
	return diff
}

// Size returns the number of unique elements in the set
//
// # Use when you need to know how many distinct elements are present
//
// Example: count := mySet.Size() // Returns 3 for a set containing [1,2,3]
func (s *Set[T]) Size() int {
	return s.m.Size()
}

// IsEmpty checks if the set has no elements
//
// # Use when you need to verify if a set contains any elements
//
// Example: if mySet.IsEmpty() { fmt.Println("Set has no elements") }
func (s *Set[T]) IsEmpty() bool {
	return s.m.IsEmpty()
}

// Clear removes all elements from the set, resetting it to empty
//
// # Use when you need to reuse an existing set by removing all its elements
//
// Example: mySet.Clear() // Set becomes empty regardless of previous content
func (s *Set[T]) Clear() {
	if s.IsReadOnly() {
		return
	}
	s.m.Clear()
}

// Values returns a slice containing all elements in the set
//
// # Use when you need to iterate through or display all set elements
//
// Example: elements := mySet.Values() // For set {1,2,3}, returns slice [1,2,3]
func (s *Set[T]) Values() []T {
	values := s.m.Keys()
	s.arr = values
	return values
}
