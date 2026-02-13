package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

func TestGenerateHashFromInput(t *testing.T) {
	tests := []struct {
		name     string
		input    bson.Raw
		expected string
		wantErr  bool
	}{
		{
			name:     "empty input returns empty string",
			input:    bson.Raw{},
			expected: "",
			wantErr:  false,
		},
		{
			name: "simple json object generates consistent hash",
			input: func() bson.Raw {
				data := map[string]interface{}{"key": "value"}
				bytes, _ := bson.Marshal(data)
				return bytes
			}(),
			expected: func() string {
				data := map[string]interface{}{"key": "value"}
				bytes, _ := bson.Marshal(data)
				return GenerateHashFromInput(bytes)
			}(),
			wantErr: false,
		},
		{
			name: "same data in different order generates same hash",
			input: func() bson.Raw {
				data := map[string]interface{}{"a": "1", "b": "2", "c": "3"}
				bytes, _ := bson.Marshal(data)
				return bytes
			}(),
			expected: func() string {
				data := map[string]interface{}{"c": "3", "b": "2", "a": "1"}
				bytes, _ := bson.Marshal(data)
				return GenerateHashFromInput(bytes)
			}(),
			wantErr: false,
		},
		{
			name: "nested objects generate consistent hash",
			input: func() bson.Raw {
				data := map[string]interface{}{
					"user": map[string]interface{}{
						"name": "test",
						"age":  30,
					},
				}
				bytes, _ := bson.Marshal(data)
				return bytes
			}(),
			expected: func() string {
				data := map[string]interface{}{
					"user": map[string]interface{}{
						"name": "test",
						"age":  30,
					},
				}
				bytes, _ := bson.Marshal(data)
				return GenerateHashFromInput(bytes)
			}(),
			wantErr: false,
		},
		{
			name: "different values generate different hashes",
			input: func() bson.Raw {
				data := map[string]interface{}{"key": "value1"}
				bytes, _ := bson.Marshal(data)
				return bytes
			}(),
			expected: func() string {
				data := map[string]interface{}{"key": "value2"}
				bytes, _ := bson.Marshal(data)
				return GenerateHashFromInput(bytes)
			}(),
			wantErr: true, // Should be different
		},
		{
			name: "json input works correctly",
			input: func() bson.Raw {
				data := map[string]interface{}{"field1": "data1", "field2": "data2"}
				bytes, _ := json.Marshal(data)
				return bytes
			}(),
			expected: func() string {
				data := map[string]interface{}{"field1": "data1", "field2": "data2"}
				bytes, _ := json.Marshal(data)
				return GenerateHashFromInput(bytes)
			}(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHashFromInput(tt.input)

			if tt.wantErr {
				// Should NOT be equal
				assert.NotEqual(t, tt.expected, result, "Expected different hashes")
			} else {
				assert.Equal(t, tt.expected, result, "Expected same hash")
			}

			// Hash should be non-empty for non-empty input
			if len(tt.input) > 0 {
				assert.NotEmpty(t, result, "Hash should not be empty for non-empty input")
				// SHA-256 hash should be 64 characters (hex encoded)
				assert.Len(t, result, 64, "SHA-256 hash should be 64 characters")
			}
		})
	}
}

func TestGenerateHashFromInput_Deterministic(t *testing.T) {
	// Test that the same input always generates the same hash
	data := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": map[string]interface{}{
			"nested": "data",
		},
	}
	bytes, err := bson.Marshal(data)
	assert.NoError(t, err)

	hash1 := GenerateHashFromInput(bytes)
	hash2 := GenerateHashFromInput(bytes)
	hash3 := GenerateHashFromInput(bytes)

	assert.Equal(t, hash1, hash2, "Same input should generate same hash")
	assert.Equal(t, hash2, hash3, "Same input should generate same hash")
}

func TestGenerateHashFromInput_OrderIndependent(t *testing.T) {
	// Test that key order doesn't affect the hash
	data1 := map[string]interface{}{
		"z": "last",
		"a": "first",
		"m": "middle",
	}

	data2 := map[string]interface{}{
		"a": "first",
		"m": "middle",
		"z": "last",
	}

	bytes1, _ := bson.Marshal(data1)
	bytes2, _ := bson.Marshal(data2)

	hash1 := GenerateHashFromInput(bytes1)
	hash2 := GenerateHashFromInput(bytes2)

	assert.Equal(t, hash1, hash2, "Different key order should generate same hash")
}

func TestDeepCopy(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]any
	}{
		{
			name:  "empty map",
			input: map[string]any{},
		},
		{
			name: "simple map",
			input: map[string]any{
				"key1": "value1",
				"key2": 123,
			},
		},
		{
			name: "nested map",
			input: map[string]any{
				"key1": "value1",
				"nested": map[string]any{
					"inner": "value",
				},
			},
		},
		{
			name: "deeply nested map",
			input: map[string]any{
				"level1": map[string]any{
					"level2": map[string]any{
						"level3": "deep",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepCopy(tt.input)

			// Check that result is not the same pointer
			if len(tt.input) > 0 {
				assert.NotSame(t, &tt.input, &result, "Deep copy should create new map")
			}

			// Check that values are equal
			assert.Equal(t, tt.input, result, "Deep copy should have same values")

			// Modify original and ensure copy is not affected
			if len(tt.input) > 0 {
				for k := range tt.input {
					tt.input[k] = "modified"
					break
				}
				assert.NotEqual(t, tt.input, result, "Modifying original should not affect copy")
			}
		})
	}
}

func BenchmarkGenerateHashFromInput(b *testing.B) {
	data := map[string]interface{}{
		"field1": "value1",
		"field2": "value2",
		"field3": map[string]interface{}{
			"nested1": "data1",
			"nested2": "data2",
		},
	}
	bytes, _ := bson.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateHashFromInput(bytes)
	}
}
