package cluster

import (
	"testing"
	"time"

	"github.com/spechtlabs/tka/pkg/cluster/messages"
	"github.com/stretchr/testify/assert"
)

func TestGossipNode_StateTransitions(t *testing.T) {
	t.Helper()

	tests := []struct {
		name          string
		initialState  messages.PeerState
		action        func(*GossipNode)
		expectedState messages.PeerState
		description   string
	}{
		{
			name:          "New node starts healthy",
			initialState:  messages.PeerState_PEER_STATE_HEALTHY,
			action:        func(n *GossipNode) {},
			expectedState: messages.PeerState_PEER_STATE_HEALTHY,
			description:   "A new node should be in healthy state",
		},
		{
			name:         "Healthy to suspected dead",
			initialState: messages.PeerState_PEER_STATE_HEALTHY,
			action: func(n *GossipNode) {
				n.MarkSuspectedDead()
			},
			expectedState: messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			description:   "Can transition from healthy to suspected dead",
		},
		{
			name:         "Suspected dead to dead",
			initialState: messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			action: func(n *GossipNode) {
				n.MarkDead()
			},
			expectedState: messages.PeerState_PEER_STATE_DEAD,
			description:   "Can transition from suspected dead to dead",
		},
		{
			name:         "Suspected dead to healthy",
			initialState: messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			action: func(n *GossipNode) {
				n.MarkHealthy()
			},
			expectedState: messages.PeerState_PEER_STATE_HEALTHY,
			description:   "Can resurrect from suspected dead to healthy",
		},
		{
			name:         "Dead to healthy via resurrection",
			initialState: messages.PeerState_PEER_STATE_DEAD,
			action: func(n *GossipNode) {
				n.MarkHealthy()
			},
			expectedState: messages.PeerState_PEER_STATE_HEALTHY,
			description:   "Can resurrect from dead to healthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			node := NewGossipNode("test-id", "localhost:8080")
			node.SetState(tt.initialState)

			tt.action(&node)

			assert.Equal(t, tt.expectedState, node.GetState(), tt.description)
		})
	}
}

func TestGossipNode_StateChecks(t *testing.T) {
	t.Helper()

	node := NewGossipNode("test-id", "localhost:8080")

	// Start healthy
	assert.True(t, node.IsHealthy())
	assert.False(t, node.IsSuspectedDead())
	assert.False(t, node.IsDead())

	// Transition to suspected dead
	node.MarkSuspectedDead()
	assert.False(t, node.IsHealthy())
	assert.True(t, node.IsSuspectedDead())
	assert.False(t, node.IsDead())

	// Transition to dead
	node.MarkDead()
	assert.False(t, node.IsHealthy())
	assert.False(t, node.IsSuspectedDead())
	assert.True(t, node.IsDead())

	// Resurrect
	node.MarkHealthy()
	assert.True(t, node.IsHealthy())
	assert.False(t, node.IsSuspectedDead())
	assert.False(t, node.IsDead())
}

func TestGossipNode_MarkHealthyResetsFailures(t *testing.T) {
	t.Helper()

	node := NewGossipNode("test-id", "localhost:8080")

	// Add failures
	for i := 0; i < 5; i++ {
		node.IncrementFailureCount()
	}

	node.MarkSuspectedDead()

	assert.Equal(t, 5, node.GetConsecutiveFails())
	assert.True(t, node.IsSuspectedDead())

	// Marking healthy should reset failures
	node.MarkHealthy()

	assert.Equal(t, 0, node.GetConsecutiveFails())
	assert.True(t, node.IsHealthy())
}

func TestInMemoryGossipStore_MarkPeerSuspectedDead(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080")

	// Add a peer
	store.Heartbeat("peer-1", "localhost:8081")

	peer := store.GetPeer("peer-1")
	assert.NotNil(t, peer)
	assert.True(t, peer.IsHealthy())

	// Mark peer as suspected dead
	store.MarkPeerSuspectedDead("peer-1")

	peer = store.GetPeer("peer-1")
	assert.NotNil(t, peer)
	assert.True(t, peer.IsSuspectedDead())
}

func TestInMemoryGossipStore_ResurrectPeer(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080")

	// Add a peer and mark it suspected dead
	store.Heartbeat("peer-1", "localhost:8081")
	store.MarkPeerSuspectedDead("peer-1")

	peer := store.GetPeer("peer-1")
	assert.NotNil(t, peer)
	assert.True(t, peer.IsSuspectedDead())

	// Resurrect the peer
	store.ResurrectPeer("peer-1")

	peer = store.GetPeer("peer-1")
	assert.NotNil(t, peer)
	assert.True(t, peer.IsHealthy())
	assert.Equal(t, 0, peer.GetConsecutiveFails())
}

func TestInMemoryGossipStore_RemoveStalePeers_WithStates(t *testing.T) {
	t.Helper()

	tests := []struct {
		name                 string
		setupFunc            func(GossipStore[SerializableString])
		threshold            int
		expectedRemovedCount int
		description          string
	}{
		{
			name: "Remove dead peers immediately",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")
				store.Heartbeat("peer-2", "localhost:8082")

				// Mark peer-1 as dead
				store.MarkPeerDead("peer-1")
			},
			threshold:            5,
			expectedRemovedCount: 1,
			description:          "Dead peers should be removed immediately",
		},
		{
			name: "Mark suspected dead as dead when threshold exceeded",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")

				// Mark as suspected dead and exceed threshold
				store.MarkPeerSuspectedDead("peer-1")
				for i := 0; i < 5; i++ {
					store.IncrementPeerFailure("peer-1", 5)
				}
			},
			threshold:            5,
			expectedRemovedCount: 0, // Not removed yet, just marked as dead
			description:          "Suspected dead peers exceeding threshold should be marked as dead",
		},
		{
			name: "Do not remove suspected dead below threshold",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")

				// Mark as suspected dead but don't exceed threshold
				store.MarkPeerSuspectedDead("peer-1")
				for i := 0; i < 3; i++ {
					store.IncrementPeerFailure("peer-1", 5)
				}
			},
			threshold:            5,
			expectedRemovedCount: 0,
			description:          "Suspected dead peers below threshold should not be affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			store := NewInMemoryGossipStore[SerializableString]("localhost:8080")
			tt.setupFunc(store)

			removed := store.RemoveStalePeers(tt.threshold)

			assert.Equal(t, tt.expectedRemovedCount, len(removed), tt.description)
		})
	}
}

func TestInMemoryGossipStore_Digest_ExcludesSuspectedAndDeadPeers(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080",
		WithLocalState(SerializableString("local-state")),
	)

	// Add healthy peer with state
	store.Heartbeat("peer-healthy", "localhost:8081")
	state1 := NewLastWriteWinsState(SerializableString("peer1-state"))
	data1, _ := state1.GetData().Marshal()
	store.Apply(GossipDiff{
		"peer-healthy": &messages.GossipVersionedState{
			DigestEntry: &messages.DigestEntry{
				Version:          1,
				Address:          "localhost:8081",
				LastSeenUnixNano: 0,
				PeerState:        messages.PeerState_PEER_STATE_HEALTHY,
			},
			Data: data1,
		},
	})

	// Add suspected dead peer with state
	store.Heartbeat("peer-suspected", "localhost:8082")
	state2 := NewLastWriteWinsState(SerializableString("peer2-state"))
	data2, _ := state2.GetData().Marshal()
	store.Apply(GossipDiff{
		"peer-suspected": &messages.GossipVersionedState{
			DigestEntry: &messages.DigestEntry{
				Version:          1,
				Address:          "localhost:8082",
				LastSeenUnixNano: 0,
				PeerState:        messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			},
			Data: data2,
		},
	})

	// Add dead peer with state
	store.Heartbeat("peer-dead", "localhost:8083")
	state3 := NewLastWriteWinsState(SerializableString("peer3-state"))
	data3, _ := state3.GetData().Marshal()
	store.Apply(GossipDiff{
		"peer-dead": &messages.GossipVersionedState{
			DigestEntry: &messages.DigestEntry{
				Version:          1,
				Address:          "localhost:8083",
				LastSeenUnixNano: 0,
				PeerState:        messages.PeerState_PEER_STATE_DEAD,
			},
			Data: data3,
		},
	})

	// Get digest
	digest, errors := store.Digest()

	assert.Equal(t, 0, len(errors))

	// Should include local node and healthy peer only
	assert.Contains(t, digest, store.GetId(), "Digest should include local node")
	assert.Contains(t, digest, "peer-healthy", "Digest should include healthy peer")

	// Should NOT include suspected dead or dead peers
	assert.NotContains(t, digest, "peer-suspected", "Digest should not include suspected dead peer")
	assert.NotContains(t, digest, "peer-dead", "Digest should not include dead peer")
}

func TestInMemoryGossipStore_IncrementPeerFailure_AutoMarksSuspected(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080")

	// Add a peer
	store.Heartbeat("peer-1", "localhost:8081")

	peer := store.GetPeer("peer-1")
	assert.True(t, peer.IsHealthy())

	// Single failure should mark as suspected dead (threshold=1)
	store.IncrementPeerFailure("peer-1", 1)

	peer = store.GetPeer("peer-1")
	assert.True(t, peer.IsSuspectedDead(), "Peer should be marked suspected dead after first failure")
}

func TestInMemoryGossipStore_Apply_HandlesStateTransitions(t *testing.T) {
	t.Helper()

	tests := []struct {
		name          string
		setupFunc     func(GossipStore[SerializableString])
		diff          GossipDiff
		expectedState messages.PeerState
		description   string
	}{
		{
			name: "Suspected dead peer marked locally when remote suspects",
			setupFunc: func(store GossipStore[SerializableString]) {
				// Add peer with initial state
				store.Heartbeat("peer-1", "localhost:8081")
				state := NewLastWriteWinsState(SerializableString("peer1-state"))
				data, _ := state.GetData().Marshal()
				store.Apply(GossipDiff{
					"peer-1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "localhost:8081",
							LastSeenUnixNano: 0,
							PeerState:        messages.PeerState_PEER_STATE_HEALTHY,
						},
						Data: data,
					},
				})
			},
			diff: func() GossipDiff {
				state := NewLastWriteWinsState(SerializableString("peer1-state-updated"))
				data, _ := state.GetData().Marshal()
				return GossipDiff{
					"peer-1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          2,
							Address:          "localhost:8081",
							LastSeenUnixNano: 0,
							PeerState:        messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
						},
						Data: data,
					},
				}
			}(),
			expectedState: messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			description:   "Peer should be marked suspected dead when remote suspects",
		},
		{
			name: "Suspected dead peer resurrected when remote has significantly newer info",
			setupFunc: func(store GossipStore[SerializableString]) {
				// No-op: we'll create a store with a short resurrection threshold in the test
			},
			diff: func() GossipDiff {
				state := NewLastWriteWinsState(SerializableString("peer1-state-updated"))
				data, _ := state.GetData().Marshal()
				// Remote has seen this peer 10 seconds in the future (well beyond threshold)
				futureTime := time.Now().Add(10 * time.Second)
				return GossipDiff{
					"peer-1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          2,
							Address:          "localhost:8081",
							LastSeenUnixNano: futureTime.UnixNano(),
							PeerState:        messages.PeerState_PEER_STATE_HEALTHY,
						},
						Data: data,
					},
				}
			}(),
			expectedState: messages.PeerState_PEER_STATE_HEALTHY,
			description:   "Suspected dead peer should be resurrected when remote has significantly newer healthy info",
		},
		{
			name: "Ignore remote state when our info is newer",
			setupFunc: func(store GossipStore[SerializableString]) {
				// Add peer and mark it as suspected dead with current timestamp
				store.Heartbeat("peer-1", "localhost:8081")
				store.MarkPeerSuspectedDead("peer-1")
			},
			diff: func() GossipDiff {
				state := NewLastWriteWinsState(SerializableString("peer1-state-updated"))
				data, _ := state.GetData().Marshal()
				// Remote has older information (1 hour ago) saying peer is healthy
				oldTime := time.Now().Add(-1 * time.Hour)
				return GossipDiff{
					"peer-1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          2,
							Address:          "localhost:8081",
							LastSeenUnixNano: oldTime.UnixNano(),
							PeerState:        messages.PeerState_PEER_STATE_HEALTHY,
						},
						Data: data,
					},
				}
			}(),
			expectedState: messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			description:   "Should ignore remote healthy state when our info is more recent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			store := NewInMemoryGossipStore[SerializableString]("localhost:8080",
				WithLocalState(SerializableString("local-state")),
				WithResurrectionThreshold[SerializableString](1*time.Second), // Short threshold for testing
			)

			// For the resurrection test, we need to add the peer first and mark it suspected dead
			if tt.name == "Suspected dead peer resurrected when remote has significantly newer info" {
				store.Heartbeat("peer-1", "localhost:8081")
				store.MarkPeerSuspectedDead("peer-1")
			} else {
				tt.setupFunc(store)
			}

			errors := store.Apply(tt.diff)
			assert.Equal(t, 0, len(errors), "Apply should not return errors")

			peer := store.GetPeer("peer-1")
			assert.NotNil(t, peer)
			assert.Equal(t, tt.expectedState, peer.GetState(), tt.description)
		})
	}
}

func TestInMemoryGossipStore_NewPeerWithState(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080",
		WithLocalState(SerializableString("local-state")),
	)

	// Apply a new peer with healthy state - should be accepted
	state := NewLastWriteWinsState(SerializableString("peer-new-state"))
	data, _ := state.GetData().Marshal()
	diff := GossipDiff{
		"peer-new": &messages.GossipVersionedState{
			DigestEntry: &messages.DigestEntry{
				Version:          1,
				Address:          "localhost:8081",
				LastSeenUnixNano: 0,
				PeerState:        messages.PeerState_PEER_STATE_HEALTHY,
			},
			Data: data,
		},
	}

	errors := store.Apply(diff)
	assert.Equal(t, 0, len(errors))

	peer := store.GetPeer("peer-new")
	assert.NotNil(t, peer)
	assert.True(t, peer.IsHealthy(), "New peer should be added with healthy state")
}

func TestDigestEntry_IncludesPeerState(t *testing.T) {
	t.Helper()

	node := NewGossipNode("test-id", "localhost:8080")
	node.MarkSuspectedDead()

	digestEntry, err := NewDigestEntry(1, &node)

	assert.Nil(t, err)
	assert.NotNil(t, digestEntry)
	assert.Equal(t, messages.PeerState_PEER_STATE_SUSPECTED_DEAD, digestEntry.PeerState)
}

func TestInMemoryGossipStore_RejectNonHealthyNewPeers(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		peerState   messages.PeerState
		shouldAdd   bool
		description string
	}{
		{
			name:        "Accept healthy new peer",
			peerState:   messages.PeerState_PEER_STATE_HEALTHY,
			shouldAdd:   true,
			description: "Healthy peers should be added to the database",
		},
		{
			name:        "Reject suspected dead new peer",
			peerState:   messages.PeerState_PEER_STATE_SUSPECTED_DEAD,
			shouldAdd:   false,
			description: "Suspected dead peers should not be added to prevent re-adding removed peers",
		},
		{
			name:        "Reject dead new peer",
			peerState:   messages.PeerState_PEER_STATE_DEAD,
			shouldAdd:   false,
			description: "Dead peers should not be added to prevent re-adding removed peers",
		},
		{
			name:        "Accept unspecified state new peer",
			peerState:   messages.PeerState_PEER_STATE_UNSPECIFIED,
			shouldAdd:   true,
			description: "Unspecified state defaults to healthy and should be accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			store := NewInMemoryGossipStore[SerializableString]("localhost:8080",
				WithLocalState(SerializableString("local-state")),
			)

			// Try to apply a new peer with the specified state
			state := NewLastWriteWinsState(SerializableString("peer1-state"))
			data, _ := state.GetData().Marshal()

			errors := store.Apply(GossipDiff{
				"peer-1": &messages.GossipVersionedState{
					DigestEntry: &messages.DigestEntry{
						Version:          1,
						Address:          "localhost:8081",
						LastSeenUnixNano: time.Now().UnixNano(),
						PeerState:        tt.peerState,
					},
					Data: data,
				},
			})

			assert.Equal(t, 0, len(errors), "Apply should not return errors")

			peer := store.GetPeer("peer-1")
			if tt.shouldAdd {
				assert.NotNil(t, peer, tt.description)
			} else {
				assert.Nil(t, peer, tt.description)
			}
		})
	}
}
