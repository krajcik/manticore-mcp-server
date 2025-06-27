package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistry_getStringArg(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string value",
			args:     map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "non-existing key",
			args:     map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			args:     map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
		{
			name:     "empty map",
			args:     map[string]interface{}{},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getStringArg(tt.args, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getIntArg(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected int
	}{
		{
			name:     "existing int value",
			args:     map[string]interface{}{"key": 42},
			key:      "key",
			expected: 42,
		},
		{
			name:     "existing float64 value",
			args:     map[string]interface{}{"key": 42.5},
			key:      "key",
			expected: 42,
		},
		{
			name:     "non-existing key",
			args:     map[string]interface{}{"other": 123},
			key:      "key",
			expected: 0,
		},
		{
			name:     "non-numeric value",
			args:     map[string]interface{}{"key": "not a number"},
			key:      "key",
			expected: 0,
		},
		{
			name:     "zero value",
			args:     map[string]interface{}{"key": 0},
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getIntArg(tt.args, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getBoolArg(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected bool
	}{
		{
			name:     "existing true value",
			args:     map[string]interface{}{"key": true},
			key:      "key",
			expected: true,
		},
		{
			name:     "existing false value",
			args:     map[string]interface{}{"key": false},
			key:      "key",
			expected: false,
		},
		{
			name:     "non-existing key",
			args:     map[string]interface{}{"other": true},
			key:      "key",
			expected: false,
		},
		{
			name:     "non-bool value",
			args:     map[string]interface{}{"key": "true"},
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getBoolArg(tt.args, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getStringSliceArg(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected []string
	}{
		{
			name: "existing slice with interface{} elements",
			args: map[string]interface{}{
				"key": []interface{}{"one", "two", "three"},
			},
			key:      "key",
			expected: []string{"one", "two", "three"},
		},
		{
			name: "existing []string slice",
			args: map[string]interface{}{
				"key": []string{"alpha", "beta"},
			},
			key:      "key",
			expected: []string{"alpha", "beta"},
		},
		{
			name: "slice with mixed types",
			args: map[string]interface{}{
				"key": []interface{}{"string", 123, "another"},
			},
			key:      "key",
			expected: []string{"string", "another"},
		},
		{
			name:     "non-existing key",
			args:     map[string]interface{}{"other": []string{"test"}},
			key:      "key",
			expected: nil,
		},
		{
			name:     "non-slice value",
			args:     map[string]interface{}{"key": "not a slice"},
			key:      "key",
			expected: nil,
		},
		{
			name: "empty slice",
			args: map[string]interface{}{
				"key": []interface{}{},
			},
			key:      "key",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getStringSliceArg(tt.args, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getStringIntMapArg(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		args     map[string]interface{}
		key      string
		expected map[string]int
	}{
		{
			name: "existing map with int values",
			args: map[string]interface{}{
				"key": map[string]interface{}{
					"field1": 10,
					"field2": 20,
				},
			},
			key: "key",
			expected: map[string]int{
				"field1": 10,
				"field2": 20,
			},
		},
		{
			name: "existing map with float64 values",
			args: map[string]interface{}{
				"key": map[string]interface{}{
					"field1": 10.5,
					"field2": 20.0,
				},
			},
			key: "key",
			expected: map[string]int{
				"field1": 10,
				"field2": 20,
			},
		},
		{
			name: "map with mixed types",
			args: map[string]interface{}{
				"key": map[string]interface{}{
					"field1": 10,
					"field2": "not a number",
					"field3": 30.5,
				},
			},
			key: "key",
			expected: map[string]int{
				"field1": 10,
				"field3": 30,
			},
		},
		{
			name:     "non-existing key",
			args:     map[string]interface{}{"other": map[string]interface{}{"test": 1}},
			key:      "key",
			expected: nil,
		},
		{
			name:     "non-map value",
			args:     map[string]interface{}{"key": "not a map"},
			key:      "key",
			expected: nil,
		},
		{
			name: "empty map",
			args: map[string]interface{}{
				"key": map[string]interface{}{},
			},
			key:      "key",
			expected: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getStringIntMapArg(tt.args, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getBoolFromMap(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected bool
	}{
		{
			name:     "existing true value",
			m:        map[string]interface{}{"key": true},
			key:      "key",
			expected: true,
		},
		{
			name:     "existing false value",
			m:        map[string]interface{}{"key": false},
			key:      "key",
			expected: false,
		},
		{
			name:     "non-existing key",
			m:        map[string]interface{}{"other": true},
			key:      "key",
			expected: false,
		},
		{
			name:     "non-bool value",
			m:        map[string]interface{}{"key": "true"},
			key:      "key",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getBoolFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getStringFromMap(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string value",
			m:        map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "non-existing key",
			m:        map[string]interface{}{"other": "value"},
			key:      "key",
			expected: "",
		},
		{
			name:     "non-string value",
			m:        map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getStringFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getIntFromMap(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected int
	}{
		{
			name:     "existing int value",
			m:        map[string]interface{}{"key": 42},
			key:      "key",
			expected: 42,
		},
		{
			name:     "existing float64 value",
			m:        map[string]interface{}{"key": 42.5},
			key:      "key",
			expected: 42,
		},
		{
			name:     "non-existing key",
			m:        map[string]interface{}{"other": 123},
			key:      "key",
			expected: 0,
		},
		{
			name:     "non-numeric value",
			m:        map[string]interface{}{"key": "not a number"},
			key:      "key",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getIntFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRegistry_getStringSliceFromMap(t *testing.T) {
	registry := &Registry{}

	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected []string
	}{
		{
			name: "existing slice with interface{} elements",
			m: map[string]interface{}{
				"key": []interface{}{"one", "two", "three"},
			},
			key:      "key",
			expected: []string{"one", "two", "three"},
		},
		{
			name: "existing []string slice",
			m: map[string]interface{}{
				"key": []string{"alpha", "beta"},
			},
			key:      "key",
			expected: []string{"alpha", "beta"},
		},
		{
			name: "slice with mixed types",
			m: map[string]interface{}{
				"key": []interface{}{"string", 123, "another"},
			},
			key:      "key",
			expected: []string{"string", "another"},
		},
		{
			name:     "non-existing key",
			m:        map[string]interface{}{"other": []string{"test"}},
			key:      "key",
			expected: nil,
		},
		{
			name:     "non-slice value",
			m:        map[string]interface{}{"key": "not a slice"},
			key:      "key",
			expected: nil,
		},
		{
			name: "empty slice",
			m: map[string]interface{}{
				"key": []interface{}{},
			},
			key:      "key",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.getStringSliceFromMap(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
