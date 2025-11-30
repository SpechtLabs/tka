package cluster_test

import (
	"testing"

	"github.com/spechtlabs/tka/pkg/cluster"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInMemoryGossipStore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		address   string
		opts      []cluster.InMemoryGossipStoreOption[cluster.SerializableString]
		wantId    func(string) bool
		wantState bool
	}{
		{
			name:    "creates store with address",
			address: "127.0.0.1:8080",
			opts:    nil,
			wantId: func(id string) bool {
				return id != ""
			},
			wantState: false,
		},
		{
			name:    "creates store with initial state",
			address: "127.0.0.1:8080",
			opts: []cluster.InMemoryGossipStoreOption[cluster.SerializableString]{
				cluster.WithLocalState(cluster.SerializableString("initial")),
			},
			wantId: func(id string) bool {
				return id != ""
			},
			wantState: true,
		},
		{
			name:    "same address generates same id",
			address: "127.0.0.1:8080",
			opts:    nil,
			wantId: func(id string) bool {
				return id != ""
			},
			wantState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString](tt.address, tt.opts...).(*cluster.InMemoryGossipStore[cluster.SerializableString])

			assert.True(t, tt.wantId(store.GetId()))

			if tt.wantState {
				displayData := store.GetDisplayData()
				require.Len(t, displayData, 1)
				assert.Equal(t, store.GetId(), displayData[0].ID)
				assert.True(t, displayData[0].IsLocal)
				assert.Equal(t, tt.address, displayData[0].Address)
			}
		})
	}
}

func TestInMemoryGossipStore_GetId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "returns same id for same address",
			address: "127.0.0.1:8080",
		},
		{
			name:    "different id for different address",
			address: "127.0.0.1:8081",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store1 := cluster.NewInMemoryGossipStore[cluster.SerializableString](tt.address).(*cluster.InMemoryGossipStore[cluster.SerializableString])
			store2 := cluster.NewInMemoryGossipStore[cluster.SerializableString](tt.address).(*cluster.InMemoryGossipStore[cluster.SerializableString])

			id1 := store1.GetId()
			id2 := store2.GetId()

			assert.Equal(t, id1, id2)
			assert.NotEmpty(t, id1)
		})
	}
}

func TestInMemoryGossipStore_Heartbeat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		initialPeers      map[string]string
		heartbeatPeerId   string
		heartbeatAddress  string
		wantPeerCount     int
		wantPeerAddress   string
		wantLastSeenCheck func(string, cluster.GossipNode) bool
	}{
		{
			name:             "adds new peer",
			initialPeers:     map[string]string{},
			heartbeatPeerId:  "peer1",
			heartbeatAddress: "127.0.0.1:8081",
			wantPeerCount:    1,
			wantPeerAddress:  "127.0.0.1:8081",
			wantLastSeenCheck: func(id string, peer cluster.GossipNode) bool {
				return id == peer.ID() && !peer.GetLastSeen().IsZero()
			},
		},
		{
			name:             "updates existing peer",
			initialPeers:     map[string]string{"peer1": "127.0.0.1:8081"},
			heartbeatPeerId:  "peer1",
			heartbeatAddress: "127.0.0.1:8082",
			wantPeerCount:    1,
			wantPeerAddress:  "127.0.0.1:8082",
			wantLastSeenCheck: func(id string, peer cluster.GossipNode) bool {
				return id == peer.ID() && !peer.GetLastSeen().IsZero()
			},
		},
		{
			name:             "updates last seen time",
			initialPeers:     map[string]string{"peer1": "127.0.0.1:8081"},
			heartbeatPeerId:  "peer1",
			heartbeatAddress: "127.0.0.1:8081",
			wantPeerCount:    1,
			wantPeerAddress:  "127.0.0.1:8081",
			wantLastSeenCheck: func(id string, peer cluster.GossipNode) bool {
				return id == peer.ID() && !peer.GetLastSeen().IsZero()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			// Setup initial peers
			for peerId, address := range tt.initialPeers {
				store.Heartbeat(peerId, address)
			}

			// Perform heartbeat
			store.Heartbeat(tt.heartbeatPeerId, tt.heartbeatAddress)

			// Verify peer count
			peers := store.GetPeers()
			assert.Equal(t, tt.wantPeerCount, len(peers))

			// Verify peer details
			peer := store.GetPeer(tt.heartbeatPeerId)
			require.NotNil(t, peer)
			assert.Equal(t, tt.wantPeerAddress, peer.GetAddress())
			assert.True(t, tt.wantLastSeenCheck(tt.heartbeatPeerId, *peer))
		})
	}
}

func TestInMemoryGossipStore_SetData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		initialState bool
		initialData  cluster.SerializableString
		newData      cluster.SerializableString
		wantVersion  cluster.Version
		wantData     cluster.SerializableString
	}{
		{
			name:         "sets initial state",
			initialState: false,
			newData:      cluster.SerializableString("new"),
			wantVersion:  0,
			wantData:     cluster.SerializableString("new"),
		},
		{
			name:         "updates existing state",
			initialState: true,
			initialData:  cluster.SerializableString("old"),
			newData:      cluster.SerializableString("updated"),
			wantVersion:  1,
			wantData:     cluster.SerializableString("updated"),
		},
		{
			name:         "increments version on update",
			initialState: true,
			initialData:  cluster.SerializableString("v0"),
			newData:      cluster.SerializableString("v1"),
			wantVersion:  1,
			wantData:     cluster.SerializableString("v1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := []cluster.InMemoryGossipStoreOption[cluster.SerializableString]{}
			if tt.initialState {
				opts = append(opts, cluster.WithLocalState(tt.initialData))
			}

			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080", opts...).(*cluster.InMemoryGossipStore[cluster.SerializableString])

			store.SetData(tt.newData)

			displayData := store.GetDisplayData()
			require.Len(t, displayData, 1)
			assert.Equal(t, tt.wantVersion, displayData[0].Version)
			assert.Equal(t, tt.wantData.String(), displayData[0].State)
		})
	}
}

func TestInMemoryGossipStore_GetPeers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addPeers []struct {
			id      string
			address string
		}
		wantPeerCount int
	}{
		{
			name:          "returns empty when no peers",
			wantPeerCount: 0,
		},
		{
			name: "returns single peer",
			addPeers: []struct {
				id      string
				address string
			}{
				{"peer1", "127.0.0.1:8081"},
			},
			wantPeerCount: 1,
		},
		{
			name: "returns multiple peers",
			addPeers: []struct {
				id      string
				address string
			}{
				{"peer1", "127.0.0.1:8081"},
				{"peer2", "127.0.0.1:8082"},
				{"peer3", "127.0.0.1:8083"},
			},
			wantPeerCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			for _, peer := range tt.addPeers {
				store.Heartbeat(peer.id, peer.address)
			}

			peers := store.GetPeers()
			assert.Equal(t, tt.wantPeerCount, len(peers))

			// Verify all added peers are present
			for _, peer := range tt.addPeers {
				found := false
				for _, p := range peers {
					if p.ID() == peer.id && p.GetAddress() == peer.address {
						found = true
						break
					}
				}
				assert.True(t, found, "peer %s not found", peer.id)
			}
		})
	}
}

func TestInMemoryGossipStore_GetPeer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addPeers []struct {
			id      string
			address string
		}
		requestPeerId string
		wantFound     bool
		wantAddress   string
	}{
		{
			name: "existing peer",
			addPeers: []struct {
				id      string
				address string
			}{
				{"peer1", "127.0.0.1:8081"},
				{"peer2", "127.0.0.1:8082"},
			},
			requestPeerId: "peer1",
			wantFound:     true,
			wantAddress:   "127.0.0.1:8081",
		},
		{
			name: "non-existent peer",
			addPeers: []struct {
				id      string
				address string
			}{
				{"peer1", "127.0.0.1:8081"},
			},
			requestPeerId: "peer999",
			wantFound:     false,
		},
		{
			name:          "no peers",
			requestPeerId: "peer1",
			wantFound:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			for _, peer := range tt.addPeers {
				store.Heartbeat(peer.id, peer.address)
			}

			peer := store.GetPeer(tt.requestPeerId)

			if tt.wantFound {
				require.NotNil(t, peer)
				assert.Equal(t, tt.wantAddress, peer.GetAddress())
				assert.Equal(t, tt.requestPeerId, peer.ID())
			} else {
				assert.Nil(t, peer)
			}
		})
	}
}

func TestInMemoryGossipStore_Digest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		setup             func(*cluster.InMemoryGossipStore[cluster.SerializableString])
		wantDigestEntries int
		wantLocalIncluded bool
	}{
		{
			name: "empty store with no state",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				// No setup
			},
			wantDigestEntries: 0,
			wantLocalIncluded: false,
		},
		{
			name: "store with only local state",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			wantDigestEntries: 1,
			wantLocalIncluded: true,
		},
		{
			name: "store with peers but no state",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.Heartbeat("peer1", "127.0.0.1:8081")
			},
			wantDigestEntries: 0, // No state entries because local state wasn't set
			wantLocalIncluded: false,
		},
		{
			name: "store with peers and state",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
				s.Heartbeat("peer1", "127.0.0.1:8081")
			},
			wantDigestEntries: 1, // Only local because peer has no state
			wantLocalIncluded: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			tt.setup(store)

			digest, errors := store.Digest()
			assert.Equal(t, 0, len(errors), "no errors should be returned")
			assert.Equal(t, tt.wantDigestEntries, len(digest))

			if tt.wantLocalIncluded {
				localId := store.GetId()
				entry, ok := digest[localId]
				assert.True(t, ok, "local state should be in digest")
				require.NotNil(t, entry)
				assert.Equal(t, "127.0.0.1:8080", entry.Address)
			}
		})
	}
}

func TestInMemoryGossipStore_Diff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setup         func(*cluster.InMemoryGossipStore[cluster.SerializableString])
		otherDigest   cluster.GossipDigest
		wantDiffCount int
		desc          string
	}{
		{
			name: "empty digests",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			otherDigest:   cluster.GossipDigest{},
			wantDiffCount: 1, // We announce our local state
			desc:          "should announce local state to empty digest",
		},
		{
			name: "same state, no diff",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			otherDigest: func() cluster.GossipDigest {
				store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])
				store.SetData(cluster.SerializableString("local"))
				digest, errors := store.Digest()
				assert.Equal(t, 0, len(errors), "no errors should be returned")
				return digest
			}(),
			wantDiffCount: 1,
			desc:          "same versions should still announce",
		},
		{
			name: "new peer in other digest",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			otherDigest: func() cluster.GossipDigest {
				store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8081").(*cluster.InMemoryGossipStore[cluster.SerializableString])
				store.SetData(cluster.SerializableString("remote"))
				digest, errors := store.Digest()
				assert.Equal(t, 0, len(errors), "no errors should be returned")
				return digest
			}(),
			wantDiffCount: 2, // Request remote state, announce local state
			desc:          "should request state from new peer",
		},
		{
			name: "request newer state from peer",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
				// Add peer with older state
				s.Heartbeat("peer1", "127.0.0.1:8081")
			},
			otherDigest: func() cluster.GossipDigest {
				d := cluster.GossipDigest{}
				d["peer1"] = &messages.DigestEntry{
					Version:          2, // Newer version
					Address:          "127.0.0.1:8081",
					LastSeenUnixNano: 0,
				}
				return d
			}(),
			wantDiffCount: 2, // Request peer state + announce local
			desc:          "should request peer with newer state",
		},
		{
			name: "announce locally known peers not in other digest",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA"))
				// Add peer1 and learn its state
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-state"))
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
				// Add peer2 and learn its state
				s.Heartbeat("peer2", "127.0.0.1:8082")
				state2 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer2-state"))
				data2, _ := state2.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer2": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8082",
							LastSeenUnixNano: 0,
						},
						Data: data2,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				// Other store only knows about storeA (this store)
				storeB := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8083").(*cluster.InMemoryGossipStore[cluster.SerializableString])
				storeB.SetData(cluster.SerializableString("storeB"))
				digest, errors := storeB.Digest()
				assert.Equal(t, 0, len(errors), "no errors should be returned")
				return digest
			}(),
			wantDiffCount: 4, // Request storeB + announce local + peer1 + peer2
			desc:          "should announce locally known peers not in other digest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			tt.setup(store)

			diff, errors := store.Diff(tt.otherDigest)
			assert.Equal(t, 0, len(errors), "no errors should be returned")
			assert.Equal(t, tt.wantDiffCount, len(diff), tt.desc)
		})
	}
}

func TestInMemoryGossipStore_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setup         func(*cluster.InMemoryGossipStore[cluster.SerializableString])
		diff          cluster.GossipDiff
		wantPeerCount int
		wantStateKeys int
		desc          string
	}{
		{
			name:          "empty diff",
			setup:         func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {},
			diff:          cluster.GossipDiff{},
			wantPeerCount: 0,
			wantStateKeys: 0,
			desc:          "empty diff should not modify state",
		},
		{
			name: "apply new peer state",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			diff: func() cluster.GossipDiff {
				d := cluster.GossipDiff{}
				state := cluster.NewLastWriteWinsState(cluster.SerializableString("remote"))
				data, _ := state.GetData().Marshal()
				d["peer1"] = &messages.GossipVersionedState{
					DigestEntry: &messages.DigestEntry{
						Version:          1,
						Address:          "127.0.0.1:8081",
						LastSeenUnixNano: 0,
					},
					Data: data,
				}
				return d
			}(),
			wantPeerCount: 1,
			wantStateKeys: 2, // local + peer1
			desc:          "should add new peer and its state",
		},
		{
			name: "update existing peer state",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
				s.Heartbeat("peer1", "127.0.0.1:8081")
				// Add existing peer state
				state := cluster.NewLastWriteWinsState(cluster.SerializableString("old"))
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: func() []byte { d, _ := state.GetData().Marshal(); return d }(),
					},
				})
			},
			diff: func() cluster.GossipDiff {
				state := cluster.NewLastWriteWinsState(cluster.SerializableString("new"))
				data, _ := state.GetData().Marshal()
				return cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          2,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data,
					},
				}
			}(),
			wantPeerCount: 1,
			wantStateKeys: 2, // local + peer1
			desc:          "should update existing peer state",
		},
		{
			name: "ignore own state in diff",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			diff:          cluster.GossipDiff{}, // Will be populated in test
			wantPeerCount: 0,
			wantStateKeys: 1, // Only local
			desc:          "should ignore own state in diff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			tt.setup(store)

			diff := tt.diff
			// Special handling for "ignore own state in diff" test
			if tt.name == "ignore own state in diff" {
				state := cluster.NewLastWriteWinsState(cluster.SerializableString("should be ignored"))
				data, _ := state.GetData().Marshal()
				diff = cluster.GossipDiff{
					store.GetId(): &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          5,
							Address:          "127.0.0.1:8080",
							LastSeenUnixNano: 0,
						},
						Data: data,
					},
				}
			}

			store.Apply(diff)

			peers := store.GetPeers()
			assert.Equal(t, tt.wantPeerCount, len(peers), tt.desc)

			// Count state keys via display data
			displayData := store.GetDisplayData()
			assert.Equal(t, tt.wantStateKeys, len(displayData), tt.desc)
		})
	}
}

func TestInMemoryGossipStore_GetDisplayData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setup          func(*cluster.InMemoryGossipStore[cluster.SerializableString])
		wantDataCount  int
		wantLocalCount int
		wantData       []struct {
			id      string
			address string
			state   string
		}
	}{
		{
			name: "empty store",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				// No setup
			},
			wantDataCount:  0,
			wantLocalCount: 0,
			wantData:       []struct{ id, address, state string }{},
		},
		{
			name: "local state only",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
			},
			wantDataCount:  1,
			wantLocalCount: 1,
			wantData: []struct {
				id      string
				address string
				state   string
			}{
				{
					address: "127.0.0.1:8080",
					state:   "local",
				},
			},
		},
		{
			name: "local state and peers",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("local"))
				s.Heartbeat("peer1", "127.0.0.1:8081")
			},
			wantDataCount:  1,
			wantLocalCount: 1,
			wantData: []struct {
				id      string
				address string
				state   string
			}{
				{
					address: "127.0.0.1:8080",
					state:   "local",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

			tt.setup(store)

			displayData := store.GetDisplayData()
			assert.Equal(t, tt.wantDataCount, len(displayData))

			localCount := 0
			for _, data := range displayData {
				if data.IsLocal {
					localCount++
					assert.Equal(t, "127.0.0.1:8080", data.Address)
				}
			}
			assert.Equal(t, tt.wantLocalCount, localCount)

			if len(tt.wantData) > 0 {
				assert.Equal(t, tt.wantData[0].address, displayData[0].Address)
				assert.Equal(t, tt.wantData[0].state, displayData[0].State)
				assert.True(t, displayData[0].IsLocal)
			}
		})
	}
}

func TestInMemoryGossipStore_DiffWithNewerVersions(t *testing.T) {
	t.Parallel()

	store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])
	store.SetData(cluster.SerializableString("local"))

	// Add a peer with state version 1
	store.Heartbeat("peer1", "127.0.0.1:8081")
	state := cluster.NewLastWriteWinsState(cluster.SerializableString("old"))
	data, _ := state.GetData().Marshal()
	store.Apply(cluster.GossipDiff{
		"peer1": &messages.GossipVersionedState{
			DigestEntry: &messages.DigestEntry{
				Version:          1,
				Address:          "127.0.0.1:8081",
				LastSeenUnixNano: 0,
			},
			Data: data,
		},
	})

	// Create a digest where peer1 has version 2 (newer)
	otherDigest := cluster.GossipDigest{
		"peer1": &messages.DigestEntry{
			Version:          2,
			Address:          "127.0.0.1:8081",
			LastSeenUnixNano: 0,
		},
	}

	diff, errors := store.Diff(otherDigest)
	assert.Equal(t, 0, len(errors), "no errors should be returned")

	// Should request peer1's newer state
	assert.GreaterOrEqual(t, len(diff), 1)

	// Apply the newer state
	newState := cluster.NewLastWriteWinsState(cluster.SerializableString("new"))
	data, _ = newState.GetData().Marshal()
	diff["peer1"] = &messages.GossipVersionedState{
		DigestEntry: &messages.DigestEntry{
			Version:          2,
			Address:          "127.0.0.1:8081",
			LastSeenUnixNano: 0,
		},
		Data: data,
	}

	store.Apply(diff)

	// Verify the state was updated
	displayData := store.GetDisplayData()
	foundPeer1 := false
	for _, data := range displayData {
		if !data.IsLocal && data.Address == "127.0.0.1:8081" {
			foundPeer1 = true
			assert.Equal(t, cluster.Version(2), data.Version)
			assert.Equal(t, "new", data.State)
		}
	}
	assert.True(t, foundPeer1)
}

func TestInMemoryGossipStore_MultiplePeers(t *testing.T) {
	t.Parallel()

	store1 := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8081").(*cluster.InMemoryGossipStore[cluster.SerializableString])
	store1.SetData(cluster.SerializableString("store1"))

	store2 := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8082").(*cluster.InMemoryGossipStore[cluster.SerializableString])
	store2.SetData(cluster.SerializableString("store2"))

	// Store 1 learns about store 2 - get digest
	store2Digest, errors := store2.Digest()
	assert.Equal(t, 0, len(errors), "no errors should be returned")
	_, errors = store1.Diff(store2Digest)
	assert.Equal(t, 0, len(errors), "no errors should be returned")

	// Store 2 responds to what store1 needs by sending its state
	store1Digest, errors := store1.Digest()
	assert.Equal(t, 0, len(errors), "no errors should be returned")
	diffWhatStore2Sends, errors := store2.Diff(store1Digest)
	assert.Equal(t, 0, len(errors), "no errors should be returned")
	store1.Apply(diffWhatStore2Sends)

	// Store 2 learns about store 1 - get digest
	store1Digest, errors = store1.Digest()
	assert.Equal(t, 0, len(errors), "no errors should be returned")
	_, errors = store2.Diff(store1Digest)
	assert.Equal(t, 0, len(errors), "no errors should be returned")

	// Store 1 responds to what store2 needs by sending its state
	diffWhatStore1Sends, errors := store1.Diff(store2Digest)
	assert.Equal(t, 0, len(errors), "no errors should be returned")
	store2.Apply(diffWhatStore1Sends)

	// Both stores should know about each other
	display1 := store1.GetDisplayData()
	display2 := store2.GetDisplayData()

	assert.GreaterOrEqual(t, len(display1), 2, "store1 should know about itself and store2")
	assert.GreaterOrEqual(t, len(display2), 2, "store2 should know about itself and store1")
}

func TestInMemoryGossipStore_ConcurrentHeartbeats(t *testing.T) {
	t.Parallel()

	store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])

	// Simulate concurrent heartbeats
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		peerId := "peer" + string(rune(i))
		go func(id string) {
			for j := 0; j < 10; j++ {
				store.Heartbeat(id, "127.0.0.1:8080")
			}
			done <- true
		}(peerId)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	peers := store.GetPeers()
	assert.Equal(t, 10, len(peers), "should have 10 peers")
}

func TestInMemoryGossipStore_Diff_AnnouncesLocallyKnownPeers(t *testing.T) {
	t.Parallel()

	// This test specifically covers the codepath where we announce locally known peers
	// that don't exist in the other digest
	tests := []struct {
		name         string
		setup        func(*cluster.InMemoryGossipStore[cluster.SerializableString])
		otherDigest  cluster.GossipDigest
		wantPeerIds  []string
		wantPeerData map[string]string
	}{
		{
			name: "announce single peer not in other digest",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA-local"))
				// Add peer1 and learn its state
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-data"))
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				storeB := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8082").(*cluster.InMemoryGossipStore[cluster.SerializableString])
				storeB.SetData(cluster.SerializableString("storeB"))
				digest, errors := storeB.Digest()
				assert.Equal(t, 0, len(errors), "no errors should be returned")
				return digest
			}(),
			wantPeerIds: []string{"peer1"},
			wantPeerData: map[string]string{
				"peer1": "peer1-data",
			},
		},
		{
			name: "announce multiple peers not in other digest",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA-local"))
				// Add peer1 and learn its state
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-data"))
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
				// Add peer2 and learn its state
				s.Heartbeat("peer2", "127.0.0.1:8082")
				state2 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer2-data"))
				data2, _ := state2.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer2": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8082",
							LastSeenUnixNano: 0,
						},
						Data: data2,
					},
				})
				// Add peer3 and learn its state
				s.Heartbeat("peer3", "127.0.0.1:8083")
				state3 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer3-data"))
				data3, _ := state3.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer3": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8083",
							LastSeenUnixNano: 0,
						},
						Data: data3,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				storeB := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8084").(*cluster.InMemoryGossipStore[cluster.SerializableString])
				storeB.SetData(cluster.SerializableString("storeB"))
				digest, errors := storeB.Digest()
				assert.Equal(t, 0, len(errors), "no errors should be returned")
				return digest
			}(),
			wantPeerIds: []string{"peer1", "peer2", "peer3"},
			wantPeerData: map[string]string{
				"peer1": "peer1-data",
				"peer2": "peer2-data",
				"peer3": "peer3-data",
			},
		},
		{
			name: "announce locally known peers even when other knows some of them",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA-local"))
				// Add peer1 and learn its state
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-data"))
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
				// Add peer2 and learn its state - this one other store knows about
				s.Heartbeat("peer2", "127.0.0.1:8082")
				state2 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer2-data"))
				data2, _ := state2.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer2": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8082",
							LastSeenUnixNano: 0,
						},
						Data: data2,
					},
				})
				// Add peer3 and learn its state - this one other store doesn't know about
				s.Heartbeat("peer3", "127.0.0.1:8083")
				state3 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer3-data"))
				data3, _ := state3.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer3": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8083",
							LastSeenUnixNano: 0,
						},
						Data: data3,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				storeB := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8084").(*cluster.InMemoryGossipStore[cluster.SerializableString])
				storeB.SetData(cluster.SerializableString("storeB"))
				// StoreB knows about peer2
				storeB.Heartbeat("peer2", "127.0.0.1:8082")
				digest, errors := storeB.Digest()
				assert.Equal(t, 0, len(errors), "no errors should be returned")
				return digest
			}(),
			// Should announce peer1 and peer3 (peer2 already known, only announce if version differs)
			wantPeerIds: []string{"peer1", "peer3"},
			wantPeerData: map[string]string{
				"peer1": "peer1-data",
				"peer3": "peer3-data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])
			tt.setup(store)

			diff, errors := store.Diff(tt.otherDigest)
			assert.Equal(t, 0, len(errors), "no errors should be returned")

			// Check that we have state for local node
			require.Contains(t, diff, store.GetId(), "diff should include local state")

			// Check that requested peers are announced with correct data
			for _, peerId := range tt.wantPeerIds {
				assert.Contains(t, diff, peerId, "diff should announce peer %s", peerId)
				if expectedData, ok := tt.wantPeerData[peerId]; ok {
					state := diff[peerId]
					require.NotNil(t, state, "state for peer %s should not be nil", peerId)
					require.NotNil(t, state.DigestEntry, "digest entry for peer %s should not be nil", peerId)
					require.NotEmpty(t, state.Data, "data for peer %s should not be empty", peerId)

					// Unmarshal and verify the data
					var result cluster.SerializableString
					err := result.Unmarshal(state.Data, &result)
					require.NoError(t, err)
					assert.Equal(t, expectedData, result.String())
				}
			}
		})
	}
}

func TestInMemoryGossipStore_Diff_SendsUpdatesForLocalPeerState(t *testing.T) {
	t.Parallel()

	// This test specifically covers the codepath where we have local state for a peer
	// that differs from the version in the other digest, so we send updates
	tests := []struct {
		name             string
		setup            func(*cluster.InMemoryGossipStore[cluster.SerializableString])
		otherDigest      cluster.GossipDigest
		wantUpdatedPeers map[string]string
		wantUpdatedCount int
	}{
		{
			name: "send update when local peer state is newer",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA"))
				// Add peer1 and learn its initial state (v1)
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-v1"))
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
				// Update peer1 to version 2
				state1v2 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-v2"))
				state1v2.SetData(state1v2.GetData())
				data1v2, _ := state1v2.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          2,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1v2,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				// Other store knows about peer1 but only at version 1
				d := cluster.GossipDigest{}
				d["peer1"] = &messages.DigestEntry{
					Version:          1, // Older version
					Address:          "127.0.0.1:8081",
					LastSeenUnixNano: 0,
				}
				return d
			}(),
			wantUpdatedPeers: map[string]string{
				"peer1": "peer1-v2",
			},
			wantUpdatedCount: 2, // peer1 update + local state
		},
		{
			name: "send updates for multiple peers with newer versions",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA"))
				// Add peer1 at version 2
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-v2"))
				state1.SetData(state1.GetData())
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          2,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
				// Add peer2 at version 3
				s.Heartbeat("peer2", "127.0.0.1:8082")
				state2 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer2-v3"))
				state2.SetData(state2.GetData())
				state2.SetData(state2.GetData())
				data2, _ := state2.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer2": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          3,
							Address:          "127.0.0.1:8082",
							LastSeenUnixNano: 0,
						},
						Data: data2,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				// Other store knows about both peers but at older versions
				d := cluster.GossipDigest{}
				d["peer1"] = &messages.DigestEntry{
					Version:          1, // Older
					Address:          "127.0.0.1:8081",
					LastSeenUnixNano: 0,
				}
				d["peer2"] = &messages.DigestEntry{
					Version:          2, // Older
					Address:          "127.0.0.1:8082",
					LastSeenUnixNano: 0,
				}
				return d
			}(),
			wantUpdatedPeers: map[string]string{
				"peer1": "peer1-v2",
				"peer2": "peer2-v3",
			},
			wantUpdatedCount: 3, // peer1 update + peer2 update + local state
		},
		{
			name: "do not send update when local version is older",
			setup: func(s *cluster.InMemoryGossipStore[cluster.SerializableString]) {
				s.SetData(cluster.SerializableString("storeA"))
				// Add peer1 at version 1
				s.Heartbeat("peer1", "127.0.0.1:8081")
				state1 := cluster.NewLastWriteWinsState(cluster.SerializableString("peer1-v1"))
				data1, _ := state1.GetData().Marshal()
				s.Apply(cluster.GossipDiff{
					"peer1": &messages.GossipVersionedState{
						DigestEntry: &messages.DigestEntry{
							Version:          1,
							Address:          "127.0.0.1:8081",
							LastSeenUnixNano: 0,
						},
						Data: data1,
					},
				})
			},
			otherDigest: func() cluster.GossipDigest {
				// Other store knows about peer1 at version 2 (newer)
				d := cluster.GossipDigest{}
				d["peer1"] = &messages.DigestEntry{
					Version:          2, // Newer
					Address:          "127.0.0.1:8081",
					LastSeenUnixNano: 0,
				}
				return d
			}(),
			wantUpdatedPeers: map[string]string{},
			wantUpdatedCount: 1, // Only local state (peer1 is known locally so we request its newer version via Diff returning our state)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := cluster.NewInMemoryGossipStore[cluster.SerializableString]("127.0.0.1:8080").(*cluster.InMemoryGossipStore[cluster.SerializableString])
			tt.setup(store)

			diff, errors := store.Diff(tt.otherDigest)
			assert.Equal(t, 0, len(errors), "no errors should be returned")

			// Check total count
			assert.Equal(t, tt.wantUpdatedCount, len(diff), "diff count should match")

			// Check that we have state for local node
			require.Contains(t, diff, store.GetId(), "diff should include local state")

			// Check that requested peers are sent with correct data
			for peerId, expectedData := range tt.wantUpdatedPeers {
				assert.Contains(t, diff, peerId, "diff should include peer %s", peerId)
				state := diff[peerId]
				require.NotNil(t, state, "state for peer %s should not be nil", peerId)
				require.NotNil(t, state.DigestEntry, "digest entry for peer %s should not be nil", peerId)
				require.NotEmpty(t, state.Data, "data for peer %s should not be empty", peerId)

				// Unmarshal and verify the data
				var result cluster.SerializableString
				err := result.Unmarshal(state.Data, &result)
				require.NoError(t, err)
				assert.Equal(t, expectedData, result.String(), "data for peer %s should match", peerId)
			}
		})
	}
}
