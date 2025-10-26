package cluster

import (
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
)

type TestGossipStore struct {
	id        string
	peersLock sync.RWMutex
	peers     map[string]GossipNodeState
}

type TestGossipStoreOption func(*TestGossipStore)

func WithLocalState(state string) TestGossipStoreOption {
	return func(s *TestGossipStore) {
		s.peersLock.Lock()
		defer s.peersLock.Unlock()

		localNodeState := s.peers[s.id]
		localNodeState.state.SetData(state)
		s.peers[s.id] = localNodeState
	}
}

func NewTestGossipStore(address string, opts ...TestGossipStoreOption) GossipStore {
	id := hashString(address)

	s := &TestGossipStore{
		id: id,
		peers: map[string]GossipNodeState{
			id: {
				node:        GossipNode{id: id, address: address, lastSeen: time.Now()},
				state:       NewLastWriteWinsState[string](""),
				lastUpdated: time.Now(),
			},
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *TestGossipStore) GetId() string {
	return s.id
}

func (s *TestGossipStore) Heartbeat(peerId string, address string) humane.Error {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	if nodeState, ok := s.peers[peerId]; ok {
		nodeState.node.Heartbeat(address)
		nodeState.lastUpdated = time.Now()
		s.peers[peerId] = nodeState
	} else {
		s.peers[peerId] = GossipNodeState{
			node:        GossipNode{id: peerId, address: address, lastSeen: time.Now()},
			state:       NewLastWriteWinsState[string](""),
			lastUpdated: time.Now(),
		}
	}

	return nil
}

func (s *TestGossipStore) SetData(status string) humane.Error {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	localNodeState := s.peers[s.id]
	localNodeState.SetData(status)
	s.peers[s.id] = localNodeState

	return nil
}

func (s *TestGossipStore) GetPeers() []GossipNode {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	peers := make([]GossipNode, 0, len(s.peers))
	for _, nodeState := range s.peers {
		if nodeState.node.id == s.id {
			continue
		}
		peers = append(peers, nodeState.node)
	}

	return peers
}

func (s *TestGossipStore) GetPeer(peerId string) GossipNode {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	if nodeState, ok := s.peers[peerId]; ok {
		return nodeState.node
	}

	return GossipNode{}
}

func (s *TestGossipStore) Digest() GossipDigest {
	digest := make(GossipDigest)
	for peerId, nodeState := range s.peers {
		digest[peerId] = &messages.DigestEntry{
			Version:          uint64(nodeState.state.GetVersion()),
			Address:          nodeState.node.address,
			LastSeenUnixNano: nodeState.node.lastSeen.UnixNano(),
		}
	}

	return digest
}

func (s *TestGossipStore) Diff(other GossipDigest) GossipDiff {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	diff := make(GossipDiff)

	// for each peer in the other digest, check if the local state is different
	// if it is, add the diff to the diff map
	// if diff contains new peers, add them to the local state with empty state
	for peerId, digest := range other {
		if peerState, ok := s.peers[peerId]; ok {
			if diffState := peerState.state.Diff(Version(digest.Version)); diffState != nil {
				diff[peerId] = &messages.GossipVersionedState{
					DigestEntry: &messages.DigestEntry{
						Version:          uint64(diffState.GetVersion()),
						Address:          peerState.node.address,
						LastSeenUnixNano: peerState.node.lastSeen.UnixNano(),
					},
					Data: []byte(diffState.GetData()),
				}
			}
		} else {
			// Peer exists in other digest but not in local state - we need to request it
			// Add it to diff with empty data to indicate we need this peer's state
			diff[peerId] = &messages.GossipVersionedState{
				DigestEntry: &messages.DigestEntry{
					Version:          0, // We don't have this peer's state yet
					Address:          digest.Address,
					LastSeenUnixNano: digest.LastSeenUnixNano,
				},
				Data: []byte(""), // Empty data indicates we need this peer's state
			}
		}
	}

	// Check for peers that exist locally but not in the other digest
	// This ensures we announce peers we know about that the other node might not know
	for peerId, peerState := range s.peers {
		if _, existsInOther := other[peerId]; !existsInOther {
			// This peer exists locally but not in the other digest
			// Add it to diff so the other node learns about it
			diff[peerId] = &messages.GossipVersionedState{
				DigestEntry: &messages.DigestEntry{
					Version:          uint64(peerState.state.GetVersion()),
					Address:          peerState.node.address,
					LastSeenUnixNano: peerState.node.lastSeen.UnixNano(),
				},
				Data: []byte(peerState.state.GetData()),
			}
		}
	}

	return diff
}

func (s *TestGossipStore) Apply(diff GossipDiff) humane.Error {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	causes := make([]error, 0, len(diff))

	// for each peer in the diff, apply the diff to the local state
	// if the peer is not in the local state, add it to the local state
	for peerId, versionState := range diff {
		if peerState, ok := s.peers[peerId]; ok {
			if err := peerState.state.Apply(&LastWriteWinsState[string]{
				version: Version(versionState.DigestEntry.Version),
				data:    string(versionState.Data),
			}); err != nil {
				causes = append(causes, err)
				continue
			}
			peerState.lastUpdated = time.Now()
			s.peers[peerId] = peerState
		} else {
			// if the peer is not in the local state, add it to the local state
			s.peers[peerId] = GossipNodeState{
				node: GossipNode{
					id:       peerId,
					address:  versionState.DigestEntry.Address,
					lastSeen: time.UnixMicro(versionState.DigestEntry.LastSeenUnixNano),
				},
				state: &LastWriteWinsState[string]{
					version: Version(versionState.DigestEntry.Version),
					data:    string(versionState.Data),
				},
				lastUpdated: time.Now(),
			}
		}
	}

	if len(causes) == 0 {
		return nil
	}

	// if there are any errors, create a long cause error with all the causes but preserve the original error context of the first error
	errLines := make([]string, 0, len(causes))
	for _, cause := range causes {
		errLines = append(errLines, cause.Error())
	}

	allErrs := humane.Wrap(causes[0], strings.Join(errLines, "\n"))

	return humane.Wrap(allErrs, "Failed to apply diff to some peers")
}

func (s *TestGossipStore) GetDisplayData() []NodeDisplayData {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	data := make([]NodeDisplayData, 0, len(s.peers))
	keys := make([]string, 0, len(s.peers))
	for k := range s.peers {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	for _, peerId := range keys {
		peerState := s.peers[peerId]
		data = append(data, NodeDisplayData{
			ID:          peerId,
			Address:     peerState.node.address,
			LastSeen:    peerState.node.lastSeen,
			Version:     peerState.state.GetVersion(),
			State:       peerState.state.GetData(),
			LastUpdated: peerState.lastUpdated,
			IsLocal:     peerId == s.id,
		})
	}

	return data
}
