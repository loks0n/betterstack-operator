package assert

import (
	"reflect"
	"testing"
)

// String verifies string fields.
func String(t testing.TB, field, actual, expected string) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s mismatch: got %q want %q", field, actual, expected)
	}
}

// Bool verifies boolean fields.
func Bool(t testing.TB, field string, actual, expected bool) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s mismatch: got %v want %v", field, actual, expected)
	}
}

// Int verifies integer fields.
func Int(t testing.TB, field string, actual, expected int) {
	t.Helper()
	if actual != expected {
		t.Fatalf("%s mismatch: got %d want %d", field, actual, expected)
	}
}

// IntPtr verifies integer pointers.
func IntPtr(t testing.TB, field string, ptr *int, expected int) {
	t.Helper()
	if ptr == nil {
		t.Fatalf("%s is nil", field)
	}
	if *ptr != expected {
		t.Fatalf("%s mismatch: got %d want %d", field, *ptr, expected)
	}
}

// StringPtr verifies string pointers.
func StringPtr(t testing.TB, field string, ptr *string, expected string) {
	t.Helper()
	if ptr == nil {
		t.Fatalf("%s is nil", field)
	}
	if *ptr != expected {
		t.Fatalf("%s mismatch: got %q want %q", field, *ptr, expected)
	}
}

// IntSlice verifies integer slices.
func IntSlice(t testing.TB, field string, actual, expected []int) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s mismatch: got %v want %v", field, actual, expected)
	}
}

// StringSlice verifies string slices.
func StringSlice(t testing.TB, field string, actual, expected []string) {
	t.Helper()
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("%s mismatch: got %v want %v", field, actual, expected)
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
