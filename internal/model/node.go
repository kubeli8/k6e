package model

type NodeStatus string

const (
	NodeStatusReady    NodeStatus = "Ready"
	NodeStatusNotReady NodeStatus = "NotReady"
)

type Node struct {
	ID      string     `json:"id"`
	Address string     `json:"address"`
	Status  NodeStatus `json:"status"`
}
