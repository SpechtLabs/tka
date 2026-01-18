package cluster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGossipNode_FailureTracking(t *testing.T) {
	t.Helper()

	tests := []struct {
		name          string
		failureCount  int
		threshold     int
		expectedStale bool
		description   string
	}{
		{
			name:          "Node with no failures is not stale",
			failureCount:  0,
			threshold:     5,
			expectedStale: false,
			description:   "A node with no failures should not be considered stale",
		},
		{
			name:          "Node with failures below threshold is not stale",
			failureCount:  3,
			threshold:     5,
			expectedStale: false,
			description:   "A node with 3 failures and threshold of 5 should not be stale",
		},
		{
			name:          "Node with failures at threshold is stale",
			failureCount:  5,
			threshold:     5,
			expectedStale: true,
			description:   "A node with 5 failures and threshold of 5 should be stale",
		},
		{
			name:          "Node with failures above threshold is stale",
			failureCount:  7,
			threshold:     5,
			expectedStale: true,
			description:   "A node with 7 failures and threshold of 5 should be stale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			node := NewGossipNode("test-id", "localhost:8080")

			// Increment failures to the desired count
			for i := 0; i < tt.failureCount; i++ {
				node.IncrementFailureCount()
			}

			assert.Equal(t, tt.failureCount, node.GetConsecutiveFails(), "Failure count should match")
			assert.Equal(t, tt.expectedStale, node.IsStale(tt.threshold), tt.description)
		})
	}
}

func TestGossipNode_HeartbeatResetsFailures(t *testing.T) {
	t.Helper()

	node := NewGossipNode("test-id", "localhost:8080")

	// Add some failures
	for i := 0; i < 5; i++ {
		node.IncrementFailureCount()
	}

	assert.Equal(t, 5, node.GetConsecutiveFails(), "Node should have 5 failures")
	assert.True(t, node.IsStale(5), "Node should be stale with 5 failures at threshold 5")

	// Heartbeat should reset failures
	node.Heartbeat("localhost:8080")

	assert.Equal(t, 0, node.GetConsecutiveFails(), "Heartbeat should reset failure count to 0")
	assert.False(t, node.IsStale(5), "Node should not be stale after heartbeat")
}

func TestInMemoryGossipStore_IncrementPeerFailure(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080")

	// Add a peer
	store.Heartbeat("peer-1", "localhost:8081")

	peer := store.GetPeer("peer-1")
	assert.NotNil(t, peer, "Peer should exist")
	assert.Equal(t, 0, peer.GetConsecutiveFails(), "New peer should have 0 failures")

	// Increment failures
	store.IncrementPeerFailure("peer-1", 5)
	store.IncrementPeerFailure("peer-1", 5)

	peer = store.GetPeer("peer-1")
	assert.NotNil(t, peer, "Peer should still exist")
	assert.Equal(t, 2, peer.GetConsecutiveFails(), "Peer should have 2 failures")
}

func TestInMemoryGossipStore_RemoveStalePeers(t *testing.T) {
	t.Helper()

	tests := []struct {
		name                 string
		setupFunc            func(GossipStore[SerializableString])
		threshold            int
		expectedRemovedCount int
		expectedRemovedIDs   []string
		description          string
	}{
		{
			name: "No peers to remove",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")
				store.Heartbeat("peer-2", "localhost:8082")
			},
			threshold:            5,
			expectedRemovedCount: 0,
			expectedRemovedIDs:   []string{},
			description:          "Healthy peers should not be removed",
		},
		{
			name: "Remove one stale peer",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")
				store.Heartbeat("peer-2", "localhost:8082")

				// Make peer-1 stale
				for i := 0; i < 5; i++ {
					store.IncrementPeerFailure("peer-1", 5)
				}
			},
			threshold:            5,
			expectedRemovedCount: 1,
			expectedRemovedIDs:   []string{"peer-1"},
			description:          "One stale peer should be removed",
		},
		{
			name: "Remove multiple stale peers",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")
				store.Heartbeat("peer-2", "localhost:8082")
				store.Heartbeat("peer-3", "localhost:8083")

				// Make peer-1 and peer-2 stale
				for i := 0; i < 5; i++ {
					store.IncrementPeerFailure("peer-1", 5)
					store.IncrementPeerFailure("peer-2", 5)
				}
			},
			threshold:            5,
			expectedRemovedCount: 2,
			expectedRemovedIDs:   []string{"peer-1", "peer-2"},
			description:          "Multiple stale peers should be removed",
		},
		{
			name: "Do not remove peer below threshold",
			setupFunc: func(store GossipStore[SerializableString]) {
				store.Heartbeat("peer-1", "localhost:8081")

				// Make peer-1 have failures but below threshold
				for i := 0; i < 4; i++ {
					store.IncrementPeerFailure("peer-1", 5)
				}
			},
			threshold:            5,
			expectedRemovedCount: 0,
			expectedRemovedIDs:   []string{},
			description:          "Peers below threshold should not be removed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			store := NewInMemoryGossipStore[SerializableString]("localhost:8080")
			tt.setupFunc(store)

			// peers go through two phases:
			// 1. First RemoveStalePeers marks SUSPECTED_DEAD peers as DEAD
			// 2. Second RemoveStalePeers actually removes DEAD peers
			store.RemoveStalePeers(tt.threshold)            // First call marks as DEAD
			removed := store.RemoveStalePeers(tt.threshold) // Second call removes them

			assert.Equal(t, tt.expectedRemovedCount, len(removed), tt.description)

			// Check that the correct peers were removed
			for _, expectedID := range tt.expectedRemovedIDs {
				assert.Contains(t, removed, expectedID, "Expected peer ID should be in removed list")
				assert.Nil(t, store.GetPeer(expectedID), "Removed peer should no longer exist in store")
			}
		})
	}
}

func TestInMemoryGossipStore_NeverRemoveLocalNode(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080",
		WithLocalState(SerializableString("local-state")),
	)

	localID := store.GetId()

	// Try to increment failures on local node (shouldn't work as expected)
	// But let's verify the store never removes itself
	store.Heartbeat("peer-1", "localhost:8081")

	// Make peer-1 stale
	for i := 0; i < 10; i++ {
		store.IncrementPeerFailure("peer-1", 5)
	}

	// Add failures to see if we can increment (for local node this should not be typical)
	// But even if we could, RemoveStalePeers should never remove the local node

	removed := store.RemoveStalePeers(5)

	// Local node should never be in the removed list
	assert.NotContains(t, removed, localID, "Local node should never be removed")

	// Local node should still exist in display data
	displayData := store.GetDisplayData()
	localExists := false
	for _, data := range displayData {
		if data.ID == localID {
			localExists = true
			break
		}
	}
	assert.True(t, localExists, "Local node should still exist in display data")
}

func TestInMemoryGossipStore_HeartbeatResetsFailureCount(t *testing.T) {
	t.Helper()

	store := NewInMemoryGossipStore[SerializableString]("localhost:8080")

	// Add a peer and increment failures
	store.Heartbeat("peer-1", "localhost:8081")
	for i := 0; i < 3; i++ {
		store.IncrementPeerFailure("peer-1", 5)
	}

	peer := store.GetPeer("peer-1")
	assert.NotNil(t, peer, "Peer should exist")
	assert.Equal(t, 3, peer.GetConsecutiveFails(), "Peer should have 3 failures")

	// Heartbeat should reset the failure count
	store.Heartbeat("peer-1", "localhost:8081")

	peer = store.GetPeer("peer-1")
	assert.NotNil(t, peer, "Peer should still exist")
	assert.Equal(t, 0, peer.GetConsecutiveFails(), "Heartbeat should reset failures to 0")
}

func TestGossipNode_AddressUpdate(t *testing.T) {
	t.Helper()

	node := NewGossipNode("test-id", "localhost:8080")
	assert.Equal(t, "localhost:8080", node.GetAddress())

	// Heartbeat with new address should update address
	node.Heartbeat("localhost:9090")
	assert.Equal(t, "localhost:9090", node.GetAddress())

	// Add failures and verify heartbeat resets them while updating address
	for i := 0; i < 3; i++ {
		node.IncrementFailureCount()
	}

	node.Heartbeat("localhost:7070")
	assert.Equal(t, "localhost:7070", node.GetAddress())
	assert.Equal(t, 0, node.GetConsecutiveFails())
}

func TestGossipNode_LastSeenUpdated(t *testing.T) {
	t.Helper()

	node := NewGossipNode("test-id", "localhost:8080")
	initialLastSeen := node.GetLastSeen()

	// Wait a bit and then heartbeat
	time.Sleep(10 * time.Millisecond)
	node.Heartbeat("localhost:8080")

	updatedLastSeen := node.GetLastSeen()
	assert.True(t, updatedLastSeen.After(initialLastSeen), "LastSeen should be updated after heartbeat")
}
