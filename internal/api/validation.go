package api

import (
	"fmt"
	"strings"

	"github.com/pyd-07/k6e/internal/model"
)

// validateWorkload checks that a decoded Workload carries the required
// identity fields and a usable spec: apiVersion, kind, metadata.name,
// metadata.namespace, at least one replica, and at least one container with
// a name and image. It returns a descriptive error for the first violation,
// suitable for surfacing directly in a 400 response.
func validateWorkload(workload model.Workload) error {
	if workload.APIVersion == "" {
		return fmt.Errorf("apiversion is required")
	}

	if workload.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	if strings.TrimSpace(workload.Metadata.Name) == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if strings.TrimSpace(workload.Metadata.Namespace) == "" {
		return fmt.Errorf("metadata.namespace is required")
	}

	if workload.Spec.Replicas < 1 {
		return fmt.Errorf("spec.replicas must be a positive integer")
	}

	if len(workload.Spec.Template.Containers) == 0 {
		return fmt.Errorf("spec.template.containers must contain at least one container")
	}

	for _, container := range workload.Spec.Template.Containers {
		if strings.TrimSpace(container.Name) == "" {
			return fmt.Errorf("spec.template.containers.name is required")
		}
		if strings.TrimSpace(container.Image) == "" {
			return fmt.Errorf("spec.template.containers.image is required")
		}
	}
	return nil
}

// validateNode checks that a decoded Node registration carries a non-empty
// ID and address and a recognized status.
func validateNode(node model.Node) error {
	if strings.TrimSpace(node.ID) == "" {
		return fmt.Errorf("node ID is required")
	}
	if strings.TrimSpace(node.Address) == "" {
		return fmt.Errorf("node address is required")
	}
	if !model.IsValidNodeStatus(node.Status) {
		return fmt.Errorf("node status must be either 'Ready' or 'NotReady'")
	}
	return nil
}
