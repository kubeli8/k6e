package model

import (
	"crypto/sha256"
	"encoding/json"
)

// TemplateHash returns a deterministic SHA-256 hash of the workload template,
// serialized as JSON. The hash is stored on each Assignment at creation time
// so that the controller can detect assignments that were created from an
// older version of the template (see IsAssignmentObsolete in the controller
// package).
//
// If the template cannot be marshalled to JSON, TemplateHash returns an empty
// string. Callers that compare hashes should treat an empty string as
// indicating an indeterminate result rather than equality.
func TemplateHash(template PodTemplateSpec) string {
	data, err := json.Marshal(template)
	if err != nil {
		return ""
	}

	hash := sha256.Sum256(data)

	return string(hash[:])
}
