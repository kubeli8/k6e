//go:build integration

package e2e

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kubeli8/k6e/internal/model"
	"github.com/kubeli8/k6e/internal/testutil"
)

const (
	baseURL        = "http://localhost:8080"
	namespace      = "default"
	workloadName   = "rollout-test"
	nodeID         = "node-1"
	pollInterval   = 500 * time.Millisecond
	initialTimeout = 60 * time.Second
	rolloutTimeout = 90 * time.Second
)

// TestWorkloadRollout launches the control plane and a node agent as
// subprocesses, creates a workload, and verifies that updating the workload
// template rolls the entire replica set over to containers built from the new
// template while the original containers are no longer represented by Running
// assignments.
func TestWorkloadRollout(t *testing.T) {
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("failed to determine repository root: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "k6e.db")

	controlPlaneBinary := testutil.BuildBinary(t, repoRoot, "./cmd/control-plane")
	agentBinary := testutil.BuildBinary(t, repoRoot, "./cmd/agent")

	controlPlane := testutil.StartProcess(t, repoRoot, controlPlaneBinary, "--db", dbPath)
	agent := testutil.StartProcess(t, repoRoot, agentBinary)

	client := &http.Client{Timeout: 2 * time.Second}

	// On failure, report the observable assignment state and both subprocess
	// logs. This runs before the process cleanups registered above so the log
	// files still exist and the control plane can still answer queries.
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		dumpAssignmentState(t, client)
		t.Log("control-plane process logs:")
		controlPlane.DumpLogs(t)
		t.Log("agent process logs:")
		agent.DumpLogs(t)
	})

	testutil.WaitForReadyNode(t, client, baseURL, nodeID)

	workload := testutil.Workload(workloadName, namespace)
	desiredReplicas := int(workload.Spec.Replicas)
	if desiredReplicas < 1 {
		t.Fatalf("workload fixture must request at least one replica, got %d", desiredReplicas)
	}

	initialHash := model.TemplateHash(workload.Spec.Template)
	assertValidTemplateHash(t, "initial workload template", initialHash)

	testutil.DoJSON[model.Workload](
		t,
		client,
		http.MethodPost,
		baseURL+"/api/v1/workloads",
		workload,
	)

	// Capture the complete initial replica set before touching the template.
	initial := waitForRunningReplicas(
		t,
		client,
		initialTimeout,
		fmt.Sprintf("initial replica set (%d Running assignments with template hash %s)", desiredReplicas, initialHash),
		desiredReplicas,
		initialHash,
	)

	originalContainers := make(map[string]string, len(initial))
	originalAssignmentIDs := make(map[string]bool, len(initial))
	for _, assignment := range initial {
		originalContainers[assignment.ContainerID] = assignment.ID
		originalAssignmentIDs[assignment.ID] = true
	}

	updatedWorkload := workload
	updatedWorkload.Spec.Template.Containers = append(
		[]model.ContainerSpec(nil),
		workload.Spec.Template.Containers...,
	)
	updatedWorkload.Spec.Template.Containers[0].Name += "-updated"
	expectedHash := model.TemplateHash(updatedWorkload.Spec.Template)

	if expectedHash == initialHash {
		t.Fatalf(
			"expected updated workload template to have a different hash than the initial template, both are %s",
			expectedHash,
		)
	}
	assertValidTemplateHash(t, "updated workload template", expectedHash)

	testutil.DoJSON[model.Workload](
		t,
		client,
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/workloads/%s/%s", baseURL, namespace, workloadName),
		updatedWorkload,
	)

	rolled := waitForAssignments(
		t,
		client,
		rolloutTimeout,
		fmt.Sprintf("rollout of template hash %s", expectedHash),
		func(assignments []model.Assignment) bool {
			running := runningAssignments(assignments)
			if len(running) != desiredReplicas {
				return false
			}
			for _, assignment := range running {
				if assignment.TemplateHash != expectedHash || assignment.ContainerID == "" {
					return false
				}
				if _, stale := originalContainers[assignment.ContainerID]; stale {
					return false
				}
			}
			return true
		},
	)

	// Verify the stored workload, not just the payload the test submitted.
	stored := testutil.DoJSON[model.Workload](
		t,
		client,
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/workloads/%s/%s", baseURL, namespace, workloadName),
		nil,
	)
	if storedHash := model.TemplateHash(stored.Spec.Template); storedHash != expectedHash {
		t.Fatalf(
			"stored workload has template hash %s, expected the updated hash %s",
			storedHash,
			expectedHash,
		)
	}

	running := runningAssignments(rolled)
	if len(running) != int(stored.Spec.Replicas) {
		t.Fatalf(
			"expected %d Running assignments (workload desired replicas), got %d:\n%s",
			stored.Spec.Replicas,
			len(running),
			formatAssignments(rolled),
		)
	}

	seenContainers := make(map[string]bool, len(running))
	for _, assignment := range running {
		if assignment.TemplateHash != expectedHash {
			t.Fatalf(
				"Running assignment %s has template hash %s, expected updated hash %s:\n%s",
				assignment.ID,
				assignment.TemplateHash,
				expectedHash,
				formatAssignments(rolled),
			)
		}
		if assignment.ContainerID == "" {
			t.Fatalf(
				"Running assignment %s has an empty container ID:\n%s",
				assignment.ID,
				formatAssignments(rolled),
			)
		}
		if _, stale := originalContainers[assignment.ContainerID]; stale {
			t.Fatalf(
				"Running assignment %s still uses original container %s:\n%s",
				assignment.ID,
				assignment.ContainerID,
				formatAssignments(rolled),
			)
		}
		if seenContainers[assignment.ContainerID] {
			t.Fatalf(
				"container %s is used by more than one Running assignment:\n%s",
				assignment.ContainerID,
				formatAssignments(rolled),
			)
		}
		seenContainers[assignment.ContainerID] = true
	}

	// Historical assignment records may survive the rollout, but none of the
	// original assignments may still be Running.
	for _, assignment := range rolled {
		if originalAssignmentIDs[assignment.ID] && assignment.Status == model.AssignmentStatusRunning {
			t.Fatalf(
				"original assignment %s (container %s) is still Running after the rollout:\n%s",
				assignment.ID,
				assignment.ContainerID,
				formatAssignments(rolled),
			)
		}
	}

	t.Logf(
		"rolled out %d replicas with template hash %s (original containers: %s):\n%s",
		len(running),
		expectedHash,
		strings.Join(originalContainerIDs(originalContainers), ", "),
		formatAssignments(rolled),
	)
}

// waitForAssignments polls the workload's assignments until condition holds,
// failing the calling test with the last-seen assignment state when the
// timeout elapses or the control plane returns an undecodable response.
func waitForAssignments(
	t *testing.T,
	client *http.Client,
	timeout time.Duration,
	description string,
	condition func(assignments []model.Assignment) bool,
) []model.Assignment {
	t.Helper()

	var latest []model.Assignment

	err := testutil.WaitUntil(timeout, pollInterval, func() (bool, error) {
		assignments, ok, err := testutil.GetJSON[[]model.Assignment](client, assignmentsURL())
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		latest = assignments
		return condition(assignments), nil
	})
	if err != nil {
		t.Fatalf(
			"timed out waiting for %s: %v\nlast seen assignments:\n%s",
			description,
			err,
			formatAssignments(latest),
		)
	}

	return latest
}

// waitForRunningReplicas waits until exactly want assignments are Running with
// templateHash and a non-empty container ID, verifies the container IDs are
// distinct, and returns the Running assignments.
func waitForRunningReplicas(
	t *testing.T,
	client *http.Client,
	timeout time.Duration,
	description string,
	want int,
	templateHash string,
) []model.Assignment {
	t.Helper()

	assignments := waitForAssignments(t, client, timeout, description, func(assignments []model.Assignment) bool {
		running := runningAssignments(assignments)
		if len(running) != want {
			return false
		}
		for _, assignment := range running {
			if assignment.TemplateHash != templateHash || assignment.ContainerID == "" {
				return false
			}
		}
		return true
	})

	running := runningAssignments(assignments)
	seen := make(map[string]bool, len(running))
	for _, assignment := range running {
		if seen[assignment.ContainerID] {
			t.Fatalf(
				"container %s is used by more than one Running assignment:\n%s",
				assignment.ContainerID,
				formatAssignments(assignments),
			)
		}
		seen[assignment.ContainerID] = true
	}

	return running
}

// dumpAssignmentState logs the current assignment records for the workload,
// best effort: it is called from a cleanup that may run after the control
// plane has already failed.
func dumpAssignmentState(t *testing.T, client *http.Client) {
	t.Helper()

	assignments, ok, err := testutil.GetJSON[[]model.Assignment](client, assignmentsURL())
	if err != nil || !ok {
		t.Logf("failed to fetch assignments for diagnostics: ok=%v err=%v", ok, err)
		return
	}

	t.Logf("assignments for %s/%s:\n%s", namespace, workloadName, formatAssignments(assignments))
}

func assignmentsURL() string {
	return fmt.Sprintf("%s/api/v1/assignments/%s/%s", baseURL, namespace, workloadName)
}

func runningAssignments(assignments []model.Assignment) []model.Assignment {
	var running []model.Assignment
	for _, assignment := range assignments {
		if assignment.Status == model.AssignmentStatusRunning {
			running = append(running, assignment)
		}
	}
	return running
}

func originalContainerIDs(originalContainers map[string]string) []string {
	containerIDs := make([]string, 0, len(originalContainers))
	for containerID := range originalContainers {
		containerIDs = append(containerIDs, containerID)
	}
	return containerIDs
}

func formatAssignments(assignments []model.Assignment) string {
	if len(assignments) == 0 {
		return "(none)"
	}

	lines := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		lines = append(lines, fmt.Sprintf(
			"id=%s status=%s node=%s container=%s template=%s",
			assignment.ID,
			assignment.Status,
			assignment.NodeID,
			assignment.ContainerID,
			assignment.TemplateHash,
		))
	}
	return strings.Join(lines, "\n")
}

func assertValidTemplateHash(t *testing.T, label, hash string) {
	t.Helper()

	if len(hash) != 64 {
		t.Fatalf(
			"%s: expected 64-character SHA-256 template hash, got length=%d value=%q",
			label,
			len(hash),
			hash,
		)
	}

	if _, err := hex.DecodeString(hash); err != nil {
		t.Fatalf("%s: template hash is not valid hexadecimal: %v", label, err)
	}
}
