package cluster

import (
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
)

type GossipNode interface {
	fmt.Stringer
	ID() string
	GetAddress() string
	GetLastSeen() time.Time
}

// GossipNodeState is a GossipVersionedState that is additionally associated with a GossipNode.
type GossipNodeState[T comparable] interface {
	// GossipVersionedState is the underlying state that is versioned and can be diffed and applied.
	GossipVersionedState[T]

	// GetLastSeen returns the last seen time of the node.
	GetLastSeen() time.Time

	// GetPeerState returns the state of the peer.
	GetPeerState() string

	// GetGossipNode returns the gossip node that is associated with the state.
	GetGossipNode() GossipNode

	// Heartbeat updates the last seen time of the node.
	Heartbeat(peer GossipNode) humane.Error
}
