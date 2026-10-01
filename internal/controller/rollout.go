package controller

import "github.com/pyd-07/k6e/internal/model"

func IsAssignmentObsolete(assignment model.Assignment, currentTemplateHash string) bool {
	if assignment.Status == model.AssignmentStatusRunning {
		return false
	}
	return currentTemplateHash != assignment.TemplateHash
}
