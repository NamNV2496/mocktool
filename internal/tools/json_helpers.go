package tools

import (
	"encoding/json"
	"fmt"
)

// decodeArgs unmarshals raw JSON arguments into the provided pointer. Nil or
// empty args decode to the zero value, which is convenient for read tools that
// take no arguments.
func decodeArgs(raw json.RawMessage, out any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

// schema is a small helper to keep JSON-Schema literals compact and valid.
func schema(s string) json.RawMessage {
	// Parsed at construction time to catch typos in CI, then re-marshaled to
	// canonical form.
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic(fmt.Sprintf("tools: invalid schema literal: %v\n%s", err, s))
	}
	b, _ := json.Marshal(v)
	return b
}
