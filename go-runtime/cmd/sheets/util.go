package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// stdinCreds is the JSON shape we accept on --from-stdin so secrets
// never appear as flags (visible in `ps`).
type stdinCreds struct {
	ID      string `json:"client_id"`
	Secret  string `json:"client_secret"`
	Project string `json:"project_id"`
}

func readJSONCreds() (stdinCreds, error) {
	var b stdinCreds
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return b, fmt.Errorf("stdin: %w", err)
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		return b, fmt.Errorf("stdin json: %w", err)
	}
	return b, nil
}

func jsonUnmarshalString(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}
