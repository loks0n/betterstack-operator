package assert

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Equal reports a mismatch when actual does not equal expected.
func Equal[T comparable](t testing.TB, field string, actual, expected T) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s mismatch: got %v want %v", field, actual, expected)
	}
}

// EqualPtr ensures the pointer is non-nil and matches the expected value.
func EqualPtr[T comparable](t testing.TB, field string, ptr *T, expected T) {
	t.Helper()
	if ptr == nil {
		t.Fatalf("%s is nil", field)
	}
	if *ptr != expected {
		t.Fatalf("%s mismatch: got %v want %v", field, *ptr, expected)
	}
}

// EqualSlice ensures the two slices are deeply equal.
func EqualSlice[T any](t testing.TB, field string, actual, expected []T) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s mismatch: got %v want %v", field, actual, expected)
	}
}

// Nil asserts the value is nil.
func Nil(t testing.TB, field string, value any) {
	t.Helper()
	if !isNil(value) {
		t.Fatalf("%s expected to be nil, got %v", field, value)
	}
}

// NotNil asserts the value is not nil.
func NotNil(t testing.TB, field string, value any) {
	t.Helper()
	if isNil(value) {
		t.Fatalf("%s expected to be non-nil", field)
	}
}

// NoError asserts that err is nil.
func NoError(t testing.TB, err error, format string, args ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", fmt.Sprintf(format, args...), err)
	}
}

// Error asserts that err is non-nil.
func Error(t testing.TB, err error, format string, args ...any) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error: %s", fmt.Sprintf(format, args...))
	}
}

// ErrorIs asserts that err satisfies target via errors.Is.
func ErrorIs(t testing.TB, err, target error, format string, args ...any) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("%s: expected error %v, got %v", fmt.Sprintf(format, args...), target, err)
	}
}

// ErrorContains asserts that err contains the provided substring.
func ErrorContains(t testing.TB, err error, substr string, format string, args ...any) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q: %s", substr, fmt.Sprintf(format, args...))
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("%s: expected error to contain %q, got %q", fmt.Sprintf(format, args...), substr, err.Error())
	}
}

// Item ensures a list contains an entry matching the provided accessor function.
func Item[T any](t testing.TB, field string, items []T, wantName, wantValue string, accessor func(T) (string, string)) {
	t.Helper()
	for _, item := range items {
		name, value := accessor(item)
		if name == wantName {
			if value != wantValue {
				t.Fatalf("%s %q value mismatch: got %q want %q", field, wantName, value, wantValue)
			}
			return
		}
	}
	t.Fatalf("%s %q not found", field, wantName)
}

// Convenience wrappers for common types.
func String(t testing.TB, field, actual, expected string) {
	Equal[string](t, field, actual, expected)
}

func Bool(t testing.TB, field string, actual, expected bool) {
	Equal[bool](t, field, actual, expected)
}

func Int(t testing.TB, field string, actual, expected int) {
	Equal[int](t, field, actual, expected)
}

func IntPtr(t testing.TB, field string, ptr *int, expected int) {
	EqualPtr[int](t, field, ptr, expected)
}

func StringPtr(t testing.TB, field string, ptr *string, expected string) {
	EqualPtr[string](t, field, ptr, expected)
}

func IntSlice(t testing.TB, field string, actual, expected []int) {
	EqualSlice[int](t, field, actual, expected)
}

func StringSlice(t testing.TB, field string, actual, expected []string) {
	EqualSlice[string](t, field, actual, expected)
}

func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	}
	return false
}
