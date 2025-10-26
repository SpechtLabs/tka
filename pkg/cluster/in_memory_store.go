package cluster

import (
	"slices"
	"sync"
	"time"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
	"go.uber.org/zap"
)

type TestGossipStore struct {
	id        string
	peersLock sync.RWMutex
	peers     map[string]GossipNode
	stateLock sync.RWMutex
	state     map[string]GossipVersionedState[SerializableString]
	address   string
}

type TestGossipStoreOption func(*TestGossipStore)

// WithLocalState sets the local state of the store.
// This will lock the store for the duration of the function.
func WithLocalState(state string) TestGossipStoreOption {
	return func(s *TestGossipStore) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()

		localState, ok := s.state[s.GetId()]
		if !ok {
			localState = NewLastWriteWinsState(SerializableString(state))
		} else {
			localState.SetData(SerializableString(state))
		}

		s.state[s.GetId()] = localState
	}
}

func NewTestGossipStore(address string, opts ...TestGossipStoreOption) GossipStore {
	id := hashString(address)

	s := &TestGossipStore{
		id:      id,
		address: address,
		peers:   make(map[string]GossipNode),
		state:   make(map[string]GossipVersionedState[SerializableString]),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *TestGossipStore) GetId() string {
	return s.id
}

func (s *TestGossipStore) Heartbeat(peerId string, address string) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	node, ok := s.peers[peerId]
	if !ok {
		node = NewGossipNode(peerId, address)
	}

	node.Heartbeat(address)
	s.peers[peerId] = node
}

func (s *TestGossipStore) SetData(status string) {
	// We don't need to lock the store here because the WithLocalState function will lock the store for us
	WithLocalState(status)(s)
}

func (s *TestGossipStore) GetPeers() []GossipNode {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	peers := make([]GossipNode, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	return peers
}

func (s *TestGossipStore) GetPeer(peerId string) *GossipNode {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	if peer, ok := s.peers[peerId]; ok {
		return &peer
	}

	return nil
}

func (s *TestGossipStore) Digest() GossipDigest {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	s.stateLock.RLock()
	defer s.stateLock.RUnlock()

	digest := make(GossipDigest)
	for peerId, peerState := range s.state {

		peer, ok := s.peers[peerId]
		// If we don't have the peer in the peers map, and it's not the local node, we skip it
		if !ok && peerId != s.GetId() {
			otelzap.L().Warn("Peer not found in peers map", zap.String("peerId", peerId))
			continue
		}

		digestEntry, err := NewDigestEntry(uint64(peerState.GetVersion()), &peer)
		if err != nil {
			otelzap.L().WithError(err).Error("Failed to create digest entry")
			continue
		}

		digest[peerId] = digestEntry
	}

	return digest
}

func (s *TestGossipStore) Diff(other GossipDigest) GossipDiff {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	s.stateLock.RLock()
	defer s.stateLock.RUnlock()

	diff := make(GossipDiff)

	// for each peer in the other digest, check if the local state is different
	// if it is, add the diff to the diff map
	// if diff contains new peers, add them to the list of peers but don't add them to the state yet
	// as the state will be updated when the peer's state is received
	for peerId, digest := range other {
		peerState, ok := s.state[peerId]
		if !ok {
			// Peer is not in local state yet so we need to request it
			// Add it to diff with empty data to indicate we need this peer's state
			diff[peerId] = &messages.GossipVersionedState{
				DigestEntry: &messages.DigestEntry{
					Version:          0, // We don't have this peer's state yet
					Address:          digest.Address,
					LastSeenUnixNano: digest.LastSeenUnixNano,
				},
				Data: []byte(""), // Empty data indicates we need this peer's state
			}
			continue
		}

		// Compare the local state of the peer with the version of the state of the peer in the digest we received
		if diffState := peerState.Diff(Version(digest.Version)); diffState != nil {
			digestEntry, err := NewDigestEntryFromPeerDigest(uint64(diffState.GetVersion()), digest)
			if err != nil {
				otelzap.L().WithError(err).Error("Failed to create digest entry")
				continue
			}

			data := diffState.GetData()
			if serializedData, err := data.Marshal(); err != nil {
				otelzap.L().WithError(err).Error("Failed to marshal state data")
			} else {
				diff[peerId] = &messages.GossipVersionedState{
					DigestEntry: digestEntry,
					Data:        serializedData,
				}
			}
		}
	}

	// Check for peers that exist locally but not in the other digest
	// This ensures we announce peers we know about that the other node might not know
	// First, check all peers in s.peers
	for peerId, peer := range s.peers {
		if _, existsInOther := other[peerId]; !existsInOther {
			peerState, ok := s.state[peerId]
			if !ok {
				otelzap.L().Error("Peer not found in local state, how is this possible?", zap.String("peerId", peerId))
				continue
			}

			// This peer exists locally but not in the other digest
			// Add it to diff so the other node learns about it
			digestEntry, err := NewDigestEntry(uint64(peerState.GetVersion()), &peer)
			if err != nil {
				otelzap.L().WithError(err).Error("Failed to create digest entry")
				continue
			}

			data := peerState.GetData()
			if serializedData, err := data.Marshal(); err != nil {
				otelzap.L().WithError(err).Error("Failed to marshal state data")
			} else {
				diff[peerId] = &messages.GossipVersionedState{
					DigestEntry: digestEntry,
					Data:        serializedData,
				}
			}
		}
	}

	// Also check our own local state - we want to announce ourselves to peers
	localState, ok := s.state[s.GetId()]
	if !ok {
		otelzap.L().Error("Local state not found in state map, how is this possible?", zap.String("peerId", s.GetId()))
		return diff
	}

	localNode := NewGossipNode(s.GetId(), s.address)
	digestEntry, err := NewDigestEntry(uint64(localState.GetVersion()), &localNode)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to create digest entry for local node")
	} else {
		data := localState.GetData()
		if serializedData, err := data.Marshal(); err != nil {
			otelzap.L().WithError(err).Error("Failed to marshal local state data")
		} else {
			diff[s.GetId()] = &messages.GossipVersionedState{
				DigestEntry: digestEntry,
				Data:        serializedData,
			}
		}
	}

	return diff
}

func (s *TestGossipStore) Apply(diff GossipDiff) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	// for each peer in the diff, apply the diff to the local state
	// if the peer is not in the local state, add it to the local state
	for peerId, versionState := range diff {
		// Skip if this is our own peer ID
		if peerId == s.GetId() {
			continue
		}

		peer, ok := s.peers[peerId]
		if !ok && peerId != s.GetId() {
			// Peer is not in the peers map, we need to add it
			s.peers[peerId] = NewGossipNode(peerId, versionState.DigestEntry.Address)

			var data SerializableString
			if err := data.Unmarshal(versionState.Data, &data); err != nil {
				otelzap.L().WithError(err).Error("Failed to unmarshal state data")
			} else {
				s.state[peerId] = &LastWriteWinsState[SerializableString]{
					version: Version(versionState.DigestEntry.Version),
					data:    data,
				}
			}

			continue
		}

		peer.Heartbeat(versionState.DigestEntry.Address)

		var data SerializableString
		if err := data.Unmarshal(versionState.Data, &data); err != nil {
			otelzap.L().WithError(err).Error("Failed to unmarshal state data")
		} else {
			s.state[peerId] = &LastWriteWinsState[SerializableString]{
				version: Version(versionState.DigestEntry.Version),
				data:    data,
			}
		}
	}
}

func (s *TestGossipStore) GetDisplayData() []NodeDisplayData {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	s.stateLock.RLock()
	defer s.stateLock.RUnlock()

	data := make([]NodeDisplayData, 0, len(s.state))
	keys := make([]string, 0, len(s.state))
	for k := range s.state {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	for _, peerId := range keys {
		peer, ok := s.peers[peerId]
		if !ok {
			otelzap.L().Error("Peer not found in peers map, how is this possible?", zap.String("peerId", peerId))
			continue
		}

		state, ok := s.state[peerId]
		if !ok {
			otelzap.L().Error("State not found in state map, how is this possible?", zap.String("peerId", peerId))
			continue
		}

		stateData := state.GetData()
		data = append(data, NodeDisplayData{
			ID:          peerId,
			Address:     peer.GetAddress(),
			LastSeen:    peer.GetLastSeen(),
			Version:     state.GetVersion(),
			State:       stateData.String(),
			LastUpdated: time.Now(),
			IsLocal:     peerId == s.GetId(),
		})
	}

	return data
}
