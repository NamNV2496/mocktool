package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

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

// HashInputConsistent hashes input consistently for both HTTP insert and gRPC
// Converts JSON/proto input → any → BSON → hash
// This ensures HTTP insert and gRPC queries use the same hashing logic
func HashInputConsistent(inputBytes []byte) string {
	if len(inputBytes) == 0 {
		return ""
	}

	fmt.Println("HashInputConsistent debug",
		"input_len", len(inputBytes),
		"input_hex", fmt.Sprintf("%x", inputBytes),
		"input_str", string(inputBytes),
	)

	// Try to unmarshal as JSON (handles both proto JSON and regular JSON)
	var inputData any
	if err := json.Unmarshal(inputBytes, &inputData); err != nil {
		fmt.Println("JSON unmarshal failed, using raw bytes", "error", err)
		// If not JSON, fall back to hashing raw bytes
		hash := GenerateHashFromInput(inputBytes)
		fmt.Println("HashInputConsistent result (raw)", "hash", hash)
		return hash
	}

	fmt.Println("JSON unmarshal success", "data", fmt.Sprintf("%#v", inputData))

	// Convert to BSON like in storage (same as insert API does)
	inputBSON, err := bson.Marshal(inputData)
	if err != nil {
		fmt.Println("BSON marshal failed", "error", err)
		// If BSON marshaling fails, hash as raw bytes
		hash := GenerateHashFromInput(inputBytes)
		fmt.Println("HashInputConsistent result (bson failed)", "hash", hash)
		return hash
	}

	fmt.Println("BSON marshal success",
		"bson_len", len(inputBSON),
		"bson_hex", fmt.Sprintf("%x", inputBSON),
	)

	// Hash the BSON (consistent hashing)
	hash := GenerateHashFromInput(inputBSON)
	fmt.Println("HashInputConsistent result (bson)", "hash", hash)
	return hash
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
