package model

import "time"

// NodeStatus describes the liveness of a registered node as observed by the
// control-plane liveness checker.
type NodeStatus string

const (
	// NodeStatusReady indicates the node is reachable and can accept new assignments.
	NodeStatusReady NodeStatus = "Ready"
	// NodeStatusNotReady indicates the node has missed its heartbeat deadline and
	// will not be selected by the scheduler until it resumes heartbeating.
	NodeStatusNotReady NodeStatus = "NotReady"
)

// Node represents a worker agent that has registered with the control plane.
// The control plane places container replicas on nodes and monitors their
// liveness via periodic heartbeats.
type Node struct {
	ID            string     `json:"id"`
	Address       string     `json:"address"`
	Status        NodeStatus `json:"status"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
}

// IsValidNodeStatus reports whether status is a recognized NodeStatus value.
func IsValidNodeStatus(status NodeStatus) bool {
	switch status {
	case NodeStatusReady, NodeStatusNotReady:
		return true
	default:
		return false
	}
}
