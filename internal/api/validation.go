package api

import (
	"fmt"
	"strings"

	"github.com/pyd-07/k6e/internal/model"
)

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
