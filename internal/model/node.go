package model

import "time"

type NodeStatus string

const (
	NodeStatusReady    NodeStatus = "Ready"
	NodeStatusNotReady NodeStatus = "NotReady"
)

type Node struct {
	ID            string     `json:"id"`
	Address       string     `json:"address"`
	Status        NodeStatus `json:"status"`
	LastHeartbeat time.Time  `json:"last_heartbeat"`
}

func IsValidNodeStatus(status NodeStatus) bool {
	switch status {
	case NodeStatusReady, NodeStatusNotReady:
		return true
	default:
		return false
	}
}
