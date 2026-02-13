package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
)

// GenerateHashFromInput generates a SHA-256 hash from input by sorting keys and hashing
// Supports both bson.Raw and json bytes
func GenerateHashFromInput(input bson.Raw) string {
	if len(input) == 0 {
		return ""
	}
	var doc map[string]any
	if err := bson.Unmarshal(input, &doc); err != nil {
		if err := json.Unmarshal(input, &doc); err != nil {
			hash := sha256.Sum256(input)
			return hex.EncodeToString(hash[:])
		}
	}

	sortedBytes, err := json.Marshal(deepCopy(doc))
	if err != nil {
		hash := sha256.Sum256(input)
		return hex.EncodeToString(hash[:])
	}

	hash := sha256.Sum256(sortedBytes)
	return hex.EncodeToString(hash[:])
}

func deepCopy(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		if nestedMap, ok := v.(map[string]any); ok {
			v = deepCopy(nestedMap)
		}
		result[k] = v
	}
	return result
}
