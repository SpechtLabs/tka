package cluster

import (
	"fmt"
	"time"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
)

type GossipNodeStateMap[T SerializableAndStringable] map[string]GossipVersionedState[T]

type GossipNode struct {
	id               string
	address          string
	lastSeen         time.Time
	consecutiveFails int
	state            messages.PeerState
}

func NewGossipNode(id string, address string) GossipNode {
	return GossipNode{
		id:       id,
		address:  address,
		lastSeen: time.Now(),
		state:    messages.PeerState_PEER_STATE_HEALTHY,
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

func (n *GossipNode) SetLastSeen(t time.Time) {
	n.lastSeen = t
}

func (n *GossipNode) String() string {
	return fmt.Sprintf("%s (%s)", n.id, n.address)
}

func (n *GossipNode) Heartbeat(address string) {
	if n.address != address {

		n.address = address
	}
	n.lastSeen = time.Now()
	n.consecutiveFails = 0 // Reset failure count on successful heartbeat

	// If the peer was suspected dead or dead, resurrect it since we're receiving a direct message
	if !n.IsHealthy() {
		previousState := n.state
		n.state = messages.PeerState_PEER_STATE_HEALTHY
		otelzap.L().Sugar().With(
			"nodeID", n.id,
			"previousState", previousState.String(),
		).Info("Peer resurrected via direct heartbeat")
	}
}

// IncrementFailureCount increments the consecutive failure count for this node.
func (n *GossipNode) IncrementFailureCount() {
	n.consecutiveFails++
}

// GetConsecutiveFails returns the number of consecutive failures for this node.
func (n *GossipNode) GetConsecutiveFails() int {
	return n.consecutiveFails
}

// IsStale checks if the node has exceeded the staleness threshold.
func (n *GossipNode) IsStale(threshold int) bool {
	return n.consecutiveFails >= threshold
}

// GetState returns the current peer state.
func (n *GossipNode) GetState() messages.PeerState {
	return n.state
}

// SetState updates the peer state.
func (n *GossipNode) SetState(state messages.PeerState) {
	n.state = state
}

// MarkSuspectedDead transitions the peer to suspected dead state.
func (n *GossipNode) MarkSuspectedDead() {
	if n.state == messages.PeerState_PEER_STATE_HEALTHY {
		n.state = messages.PeerState_PEER_STATE_SUSPECTED_DEAD
	}
}

// MarkDead transitions the peer to dead state.
func (n *GossipNode) MarkDead() {
	n.state = messages.PeerState_PEER_STATE_DEAD
}

// MarkHealthy transitions the peer to healthy state and resets failure count.
func (n *GossipNode) MarkHealthy() {
	n.state = messages.PeerState_PEER_STATE_HEALTHY
	n.consecutiveFails = 0
}

// IsHealthy returns true if the peer is in healthy state.
func (n *GossipNode) IsHealthy() bool {
	return n.state == messages.PeerState_PEER_STATE_HEALTHY
}

// IsSuspectedDead returns true if the peer is in suspected dead state.
func (n *GossipNode) IsSuspectedDead() bool {
	return n.state == messages.PeerState_PEER_STATE_SUSPECTED_DEAD
}

// IsDead returns true if the peer is in dead state.
func (n *GossipNode) IsDead() bool {
	return n.state == messages.PeerState_PEER_STATE_DEAD
}
