package controller

import "github.com/pyd-07/k6e/internal/model"

// IsAssignmentObsolete reports whether assignment was created from an older
// version of the workload template. An assignment is considered obsolete when
// it is currently Running and its recorded TemplateHash differs from
// currentTemplateHash (the hash of the workload's present template).
//
// Non-running assignments (Pending, Failed, Completed) are never considered
// obsolete because they have not successfully started a container from the
// old template, or they are already terminal.
func IsAssignmentObsolete(assignment model.Assignment, currentTemplateHash string) bool {
	if assignment.Status != model.AssignmentStatusRunning {
		return false
	}
	return currentTemplateHash != assignment.TemplateHash
}
