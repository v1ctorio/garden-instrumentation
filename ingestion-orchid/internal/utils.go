package internal

import (
	"os"
	"strings"
)

func LoadAPIKeys() map[string]struct{} {
	keys := make(map[string]struct{})
	raw := os.Getenv("INGEST_API_KEYS")

	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys[k] = struct{}{}
		}
	}

	return keys
}
