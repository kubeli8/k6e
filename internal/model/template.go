package model

import (
	"crypto/sha256"
	"encoding/json"
)

func TemplateHash(template PodTemplateSpec) string {
	// JSON serialization
	data, err := json.Marshal(template)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)

	return string(hash[:])
}
