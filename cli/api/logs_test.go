package api

import (
	"reflect"
	"testing"
)

func testConvertOffsets(t *testing.T, received []string, expected []int64) {
	converted, err := convertOffsets(received)
	if err != nil {
		t.Fatalf("unexpected error converting offsets: %s", err)
	}
	if !reflect.DeepEqual(converted, expected) {
		t.Fatalf("got %v expected %v", converted, expected)
	}
}

func testInt64sAreSorted(t *testing.T, values []int64, expected bool) {
	if int64sAreSorted(values) != expected {
		t.Fatalf("expected %v for sortedness for values: %v", expected, values)
	}
}

func testGenerateOffsets(t *testing.T, received, expected []int64) {
	converted := generateOffsets(received)
	if !reflect.DeepEqual(converted, expected) {
		t.Fatalf("unexpected error generating offsets got %v expected %v", converted, expected)
	}
}

func TestLogs_ConvertOffsets(t *testing.T) {
	testConvertOffsets(t, []string{"123", "456", "789"}, []int64{123, 456, 789})
	testConvertOffsets(t, []string{"456", "123", "789"}, []int64{456, 123, 789})

	testInt64sAreSorted(t, []int64{123, 124, 125}, true)
	testInt64sAreSorted(t, []int64{123, 125, 124}, false)
	testInt64sAreSorted(t, []int64{125, 123, 124}, false)

	testGenerateOffsets(t, []int64{456, 123, 789}, []int64{123, 124, 125})
}