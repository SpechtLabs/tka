package cluster

import (
	"fmt"
	"time"

	"github.com/spechtlabs/go-otel-utils/otelzap"
)

type GossipNodeStateMap[T SerializableAndStringable] map[string]GossipVersionedState[T]

type GossipNode struct {
	id       string
	address  string
	lastSeen time.Time
}

func NewGossipNode(id string, address string) GossipNode {
	return GossipNode{
		id:       id,
		address:  address,
		lastSeen: time.Now(),
	}
}

func (n *GossipNode) ID() string {
	return n.id
}

func (n *GossipNode) GetAddress() string {
	return n.address
}

func (n *GossipNode) GetLastSeen() time.Time {
	return n.lastSeen
}

func (n *GossipNode) String() string {
	return fmt.Sprintf("%s (%s)", n.id, n.address)
}

func (n *GossipNode) Heartbeat(address string) {
	if n.address != address {
		otelzap.L().Sugar().With(
			"old_address", n.address,
			"new_address", address,
		).Debug("Heartbeat: Node address changed")

		n.address = address
	}
	n.lastSeen = time.Now()
}
