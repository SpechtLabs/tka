package cluster

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
	"go.uber.org/zap"
)

type TestGossipStore[T SerializableAndStringable] struct {
	id                    string
	peersLock             sync.RWMutex
	peers                 map[string]GossipNode
	stateLock             sync.RWMutex
	state                 map[string]GossipVersionedState[T]
	address               string
	resurrectionThreshold time.Duration // How much newer remote info must be for resurrection
}

type TestGossipStoreOption[T SerializableAndStringable] func(*TestGossipStore[T])

// WithLocalState sets the local state of the store.
// This will lock the store for the duration of the function.
func WithLocalState[T SerializableAndStringable](state T) TestGossipStoreOption[T] {
	return func(s *TestGossipStore[T]) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()

		localState, ok := s.state[s.GetId()]
		if !ok {
			localState = NewLastWriteWinsState(state)
		} else {
			localState.SetData(state)
		}

		s.state[s.GetId()] = localState
	}
}

// WithResurrectionThreshold sets how much newer remote information must be
// before allowing a peer to be resurrected from SUSPECTED_DEAD or DEAD state.
func WithResurrectionThreshold[T SerializableAndStringable](threshold time.Duration) TestGossipStoreOption[T] {
	return func(s *TestGossipStore[T]) {
		s.resurrectionThreshold = threshold
	}
}

func NewTestGossipStore[T SerializableAndStringable](address string, opts ...TestGossipStoreOption[T]) GossipStore[T] {
	id := hashString(address)

	s := &TestGossipStore[T]{
		id:                    id,
		address:               address,
		peers:                 make(map[string]GossipNode),
		state:                 make(map[string]GossipVersionedState[T]),
		resurrectionThreshold: 5 * time.Second, // Default: remote info must be 5s newer for resurrection
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *TestGossipStore[T]) GetId() string {
	return s.id
}

func (s *TestGossipStore[T]) Heartbeat(peerID string, address string) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	node, ok := s.peers[peerID]
	if !ok {
		node = NewGossipNode(peerID, address)
	}

	node.Heartbeat(address)
	s.peers[peerID] = node
}

func (s *TestGossipStore[T]) SetData(status T) {
	// We don't need to lock the store here because the WithLocalState function will lock the store for us
	WithLocalState(status)(s)
}

func (s *TestGossipStore[T]) GetPeers() []GossipNode {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	peers := make([]GossipNode, 0, len(s.peers))
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	return peers
}

func (s *TestGossipStore[T]) GetPeer(peerID string) *GossipNode {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	if peer, ok := s.peers[peerID]; ok {
		return &peer
	}

	return nil
}

func (s *TestGossipStore[T]) IncrementPeerFailure(peerID string, threshold int) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	peer, ok := s.peers[peerID]
	if !ok {
		otelzap.L().Warn("Attempted to increment failure count for unknown peer", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
		return
	}

	peer.IncrementFailureCount()

	// Transition to suspected dead if threshold reached and currently healthy
	if peer.IsStale(threshold) && peer.IsHealthy() {
		peer.MarkSuspectedDead()
		otelzap.L().Info("Peer marked as suspected dead",
			zap.String("nodeID", s.GetId()),
			zap.String("peerID", peerID),
			zap.String("address", peer.GetAddress()),
		)
	}

	s.peers[peerID] = peer

	otelzap.L().Debug("Incremented peer failure count",
		zap.String("nodeID", s.GetId()),
		zap.String("peerID", peerID),
		zap.Int("consecutiveFails", peer.GetConsecutiveFails()),
		zap.String("state", peer.GetState().String()),
	)
}

func (s *TestGossipStore[T]) MarkPeerSuspectedDead(peerID string) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	peer, ok := s.peers[peerID]
	if !ok {
		otelzap.L().Warn("Attempted to mark unknown peer as suspected dead", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
		return
	}

	peer.MarkSuspectedDead()
	s.peers[peerID] = peer

	otelzap.L().Info("Peer marked as suspected dead",
		zap.String("nodeID", s.GetId()),
		zap.String("peerID", peerID),
		zap.String("address", peer.GetAddress()),
	)
}

func (s *TestGossipStore[T]) MarkPeerDead(peerID string) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	peer, ok := s.peers[peerID]
	if !ok {
		otelzap.L().Warn("Attempted to mark unknown peer as dead", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
		return
	}

	peer.MarkDead()
	s.peers[peerID] = peer

	otelzap.L().Info("Peer marked as dead",
		zap.String("nodeID", s.GetId()),
		zap.String("peerID", peerID),
		zap.String("address", peer.GetAddress()),
	)
}

func (s *TestGossipStore[T]) ResurrectPeer(peerID string) {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	peer, ok := s.peers[peerID]
	if !ok {
		otelzap.L().Warn("Attempted to resurrect unknown peer", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
		return
	}

	oldState := peer.GetState()
	peer.MarkHealthy()
	s.peers[peerID] = peer

	otelzap.L().Info("Peer resurrected",
		zap.String("nodeID", s.GetId()),
		zap.String("peerID", peerID),
		zap.String("address", peer.GetAddress()),
		zap.String("previousState", oldState.String()),
	)
}

func (s *TestGossipStore[T]) RemoveStalePeers(threshold int) []string {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	removed := make([]string, 0)

	for peerID, peer := range s.peers {
		// Never remove the local node
		if peerID == s.GetId() {
			continue
		}

		// Remove peers that are marked as dead
		if peer.IsDead() {
			otelzap.L().Info("Removing dead peer",
				zap.String("nodeID", s.GetId()),
				zap.String("peerID", peerID),
				zap.String("address", peer.GetAddress()),
				zap.String("state", peer.GetState().String()),
			)

			// Remove from peers map
			delete(s.peers, peerID)

			// Remove from state map
			delete(s.state, peerID)

			removed = append(removed, peerID)
			continue
		}

		// Mark suspected dead peers as dead if they exceed the threshold
		if peer.IsSuspectedDead() && peer.IsStale(threshold) {
			peer.MarkDead()
			s.peers[peerID] = peer

			otelzap.L().Info("Suspected dead peer exceeded threshold, marking as dead",
				zap.String("nodeID", s.GetId()),
				zap.String("peerID", peerID),
				zap.String("address", peer.GetAddress()),
				zap.Int("consecutiveFails", peer.GetConsecutiveFails()),
				zap.Int("threshold", threshold),
			)
		}
	}

	return removed
}

func (s *TestGossipStore[T]) Digest() (GossipDigest, []humane.Error) {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	s.stateLock.RLock()
	defer s.stateLock.RUnlock()

	digest := make(GossipDigest)
	errors := make([]humane.Error, 0)
	for peerID, peerState := range s.state {

		peer, ok := s.peers[peerID]
		// If we don't have the peer in the peers map, and it's not the local node, we skip it
		if !ok && peerID != s.GetId() {
			otelzap.L().Warn("Peer not found in peers map", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
			continue
		}

		// If this is the local node and not in peers, create a peer entry for it
		if !ok && peerID == s.GetId() {
			peer = NewGossipNode(peerID, s.address)
		}

		// Skip suspected dead and dead peers in the digest - they should not be re-announced
		// unless they resurrect themselves
		if peer.IsSuspectedDead() || peer.IsDead() {
			otelzap.L().Debug("Skipping non-healthy peer in digest",
				zap.String("nodeID", s.GetId()),
				zap.String("peerID", peerID),
				zap.String("state", peer.GetState().String()),
			)
			continue
		}

		digestEntry, err := NewDigestEntry(uint64(peerState.GetVersion()), &peer)
		if err != nil {
			errors = append(errors, humane.Wrap(err, "failed to create digest entry"))
			continue
		}

		digest[peerID] = digestEntry
	}

	return digest, errors
}

func (s *TestGossipStore[T]) Diff(other GossipDigest) (GossipDiff, []humane.Error) {
	s.peersLock.RLock()
	defer s.peersLock.RUnlock()

	s.stateLock.RLock()
	defer s.stateLock.RUnlock()

	diff := make(GossipDiff)
	errors := make([]humane.Error, 0)

	// Process peers in the remote digest - request state we don't have or send our newer version
	errors = s.processPeersInRemoteDigest(other, diff, errors)

	// Announce peers we know about but the remote node doesn't
	errors = s.announcePeersOnlyKnownLocally(other, diff, errors)

	// Announce our own state if the remote node needs it
	errors = s.announceLocalState(other, diff, errors)

	return diff, errors
}

func (s *TestGossipStore[T]) Apply(diff GossipDiff) []humane.Error {
	s.peersLock.Lock()
	defer s.peersLock.Unlock()

	s.stateLock.Lock()
	defer s.stateLock.Unlock()

	errors := make([]humane.Error, 0)

	// Apply each peer's state from the diff to our local state
	for peerID, versionState := range diff {
		// Skip if this is our own peer ID - we don't apply updates to ourselves because we are the authoritative source
		if peerID == s.GetId() {
			continue
		}

		// Skip nil versionState entries
		if versionState == nil {
			errors = append(errors, humane.New(fmt.Sprintf("versionState is nil for peer %s", peerID)))
			continue
		}

		_, peerExists := s.peers[peerID]

		// Note: State transition logic is handled in applyNewPeerState and applyExistingPeerState
		// based on timestamp comparisons and resurrection thresholds

		var err humane.Error
		if !peerExists {
			// Handle new peer we haven't seen before
			err = s.applyNewPeerState(peerID, versionState)
		} else {
			// Handle existing peer with updated state
			err = s.applyExistingPeerState(peerID, versionState)
		}

		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func (s *TestGossipStore[T]) GetDisplayData() []NodeDisplayData {
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

	for _, peerID := range keys {
		peer, ok := s.peers[peerID]
		if !ok {
			// If this is the local node, create a peer entry for it
			if peerID == s.GetId() {
				peer = NewGossipNode(peerID, s.address)
			} else {
				otelzap.L().Error("Peer not found in peers map, how is this possible?", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
				continue
			}
		}

		state, ok := s.state[peerID]
		if !ok {
			otelzap.L().Error("State not found in state map, how is this possible?", zap.String("nodeID", s.GetId()), zap.String("peerID", peerID))
			continue
		}

		stateData := state.GetData()
		data = append(data, NodeDisplayData{
			ID:          peerID,
			Address:     peer.GetAddress(),
			LastSeen:    peer.GetLastSeen(),
			Version:     state.GetVersion(),
			State:       stateData.String(),
			LastUpdated: time.Now(),
			IsLocal:     peerID == s.GetId(),
			PeerState:   peer.GetState(),
		})
	}

	return data
}

func gossipVersionedStateMessageFromDigest(digest *messages.DigestEntry) *messages.GossipVersionedState {
	return &messages.GossipVersionedState{
		DigestEntry: &messages.DigestEntry{
			Version:          0, // We don't have this peer's state yet
			Address:          digest.Address,
			LastSeenUnixNano: digest.LastSeenUnixNano,
			PeerState:        digest.PeerState,
		},
		Data: []byte(""), // Empty data indicates we need this peer's state
	}
}

func gossipVersionedStateMessageFromState[T SerializableAndStringable](diffState GossipVersionedState[T], digest *messages.DigestEntry) (*messages.GossipVersionedState, humane.Error) {
	digestEntry, err := NewDigestEntryFromPeerDigest(uint64(diffState.GetVersion()), digest)
	if err != nil {
		return nil, humane.Wrap(err, "failed to create digest entry")
	}

	data := diffState.GetData()
	serializedData, err := data.Marshal()
	if err != nil {
		return nil, humane.Wrap(err, "failed to marshal state data")
	}

	return &messages.GossipVersionedState{
		DigestEntry: digestEntry,
		Data:        serializedData,
	}, nil
}

// createGossipVersionedStateFromPeerState creates a GossipVersionedState message from local peer state and peer info.
func (s *TestGossipStore[T]) createGossipVersionedStateFromPeerState(peerID string, peerState GossipVersionedState[T], peer *GossipNode) (*messages.GossipVersionedState, humane.Error) {
	digestEntry, err := NewDigestEntry(uint64(peerState.GetVersion()), peer)
	if err != nil {
		return nil, humane.Wrap(err, "failed to create digest entry")
	}

	data := peerState.GetData()
	serializedData, err := data.Marshal()
	if err != nil {
		return nil, humane.Wrap(err, "failed to marshal state data")
	}

	return &messages.GossipVersionedState{
		DigestEntry: digestEntry,
		Data:        serializedData,
	}, nil
}

// processPeersInRemoteDigest handles peers that exist in the remote digest.
// For each peer, it either requests their state (if we don't have it) or sends our version (if we have a newer one).
func (s *TestGossipStore[T]) processPeersInRemoteDigest(other GossipDigest, diff GossipDiff, errors []humane.Error) []humane.Error {
	for peerID, digest := range other {
		if digest == nil {
			errors = append(errors, humane.New(fmt.Sprintf("digest is nil for peer %s", peerID)))
			continue
		}

		peerState, ok := s.state[peerID]
		if !ok {
			// Peer is not in local state yet so we need to request it
			// Add it to diff with empty data (Version 0). Resulting in the peer sending us his state data in the
			// apply response
			diff[peerID] = gossipVersionedStateMessageFromDigest(digest)
			continue
		}

		// Compare the local state of the peer with the version of the state of the peer in the digest we received
		diffState := peerState.Diff(Version(digest.Version))
		if diffState == nil {
			continue
		}

		gossipVersionedState, err := gossipVersionedStateMessageFromState(diffState, digest)
		if err != nil {
			errors = append(errors, humane.Wrap(err, "failed to create gossip versioned state message"))
			continue
		}

		diff[peerID] = gossipVersionedState
	}

	return errors
}

// announcePeersOnlyKnownLocally announces peers that we know about but the remote node doesn't.
// This ensures both nodes eventually learn about all peers in the cluster.
func (s *TestGossipStore[T]) announcePeersOnlyKnownLocally(other GossipDigest, diff GossipDiff, errors []humane.Error) []humane.Error {
	for peerID, peer := range s.peers {
		if _, existsInOther := other[peerID]; existsInOther {
			continue
		}

		peerState, ok := s.state[peerID]
		if !ok {
			errors = append(errors, humane.New(fmt.Sprintf("peer (%s) not found in local state, how is this possible?", peerID)))
			continue
		}

		// This peer exists locally but not in the other digest
		// Add it to diff so the other node learns about it
		gossipVersionedState, err := s.createGossipVersionedStateFromPeerState(peerID, peerState, &peer)
		if err != nil {
			errors = append(errors, humane.Wrap(err, "failed to create gossip versioned state"))
			continue
		}

		diff[peerID] = gossipVersionedState
	}

	return errors
}

// announceLocalState adds the local node's state to the diff if the remote node needs it.
// This is skipped if the remote node already has a version that's as new or newer than ours.
func (s *TestGossipStore[T]) announceLocalState(other GossipDigest, diff GossipDiff, errors []humane.Error) []humane.Error {
	// Skip if the remote node already knows about us with a version >= our current version
	if remoteDigest, existsInOther := other[s.GetId()]; existsInOther {
		if localState, ok := s.state[s.GetId()]; ok {
			if uint64(remoteDigest.Version) >= uint64(localState.GetVersion()) {
				return errors
			}
		}
	}

	localState, ok := s.state[s.GetId()]
	if !ok {
		otelzap.L().Error("Local state not found in state map, how is this possible?", zap.String("nodeID", s.GetId()), zap.String("peerID", s.GetId()))
		return errors
	}

	localNode := NewGossipNode(s.GetId(), s.address)
	gossipVersionedState, err := s.createGossipVersionedStateFromPeerState(s.GetId(), localState, &localNode)
	if err != nil {
		errors = append(errors, humane.Wrap(err, "failed to create gossip versioned state for local node"))
		return errors
	}

	diff[s.GetId()] = gossipVersionedState
	return errors
}

// unmarshalAndCreateState unmarshals the data from a GossipVersionedState and creates a LastWriteWinsState.
func (s *TestGossipStore[T]) unmarshalAndCreateState(versionState *messages.GossipVersionedState) (*LastWriteWinsState[T], humane.Error) {
	if versionState == nil {
		return nil, humane.New("versionState is nil")
	}
	if versionState.DigestEntry == nil {
		return nil, humane.New("versionState.DigestEntry is nil")
	}

	var data T
	if err := data.Unmarshal(versionState.Data, &data); err != nil {
		return nil, humane.Wrap(err, "failed to unmarshal state data")
	}

	return &LastWriteWinsState[T]{
		version: Version(versionState.DigestEntry.Version),
		data:    data,
	}, nil
}

// applyNewPeerState handles applying state for a peer we haven't seen before.
// It adds the peer to our peers map and initializes their state.
// IMPORTANT: We only accept new peers that are HEALTHY. If a peer is marked as
// SUSPECTED_DEAD or DEAD, we reject it to prevent re-adding peers we've already removed.
func (s *TestGossipStore[T]) applyNewPeerState(peerID string, versionState *messages.GossipVersionedState) humane.Error {
	if versionState == nil {
		return humane.New("versionState is nil")
	}
	if versionState.DigestEntry == nil {
		return humane.New("versionState.DigestEntry is nil")
	}

	// Only accept new peers that are healthy
	// If another node is gossiping about a suspected dead or dead peer, don't add it
	// This prevents re-adding peers we've already removed from our database
	if versionState.DigestEntry.PeerState == messages.PeerState_PEER_STATE_SUSPECTED_DEAD ||
		versionState.DigestEntry.PeerState == messages.PeerState_PEER_STATE_DEAD {
		otelzap.L().Debug("Rejecting new peer with non-healthy state",
			zap.String("nodeID", s.GetId()),
			zap.String("peerID", peerID),
			zap.String("peerState", versionState.DigestEntry.PeerState.String()),
			zap.String("address", versionState.DigestEntry.Address),
		)
		return nil // Not an error, just skip adding this peer
	}

	// Add the peer to our peers map
	newNode := NewGossipNode(peerID, versionState.DigestEntry.Address)

	// Set the peer state from the digest entry (should be HEALTHY or UNSPECIFIED)
	if versionState.DigestEntry.PeerState != messages.PeerState_PEER_STATE_UNSPECIFIED {
		newNode.SetState(versionState.DigestEntry.PeerState)
	}

	s.peers[peerID] = newNode

	otelzap.L().Debug("Added new peer to database",
		zap.String("nodeID", s.GetId()),
		zap.String("peerID", peerID),
		zap.String("address", versionState.DigestEntry.Address),
		zap.String("state", newNode.GetState().String()),
	)

	// Unmarshal and store the peer's state
	state, err := s.unmarshalAndCreateState(versionState)
	if err != nil {
		return err
	}

	s.state[peerID] = state
	return nil
}

var (
	ErrorVersionedStateFork                = humane.New("warning: current state and new state are the same version but have different data.", "this conflict will be automatically resolved by using the node-id as the tie-breaker. The node announcing the smaller node-id wins")
	ErrorVersionedStateMonotonicIncreasing = humane.New("rejected peer version", "ensure versions are monotonically increasing")
)

// applyExistingPeerState handles applying state for a peer we already know about.
// It updates the peer's state based on remote information.
// Note: We do NOT call Heartbeat here because this is indirect information from gossip.
// Direct heartbeats are handled separately in the message handlers.
func (s *TestGossipStore[T]) applyExistingPeerState(peerID string, versionState *messages.GossipVersionedState) humane.Error {
	if versionState == nil {
		return humane.New("versionState is nil")
	}
	if versionState.DigestEntry == nil {
		return humane.New("versionState.DigestEntry is nil")
	}

	peer := s.peers[peerID]

	// Update peer state from digest entry if provided
	// We apply different rules for state transitions:
	// - Health degradation (HEALTHY -> SUSPECTED/DEAD): Always apply (fast failure detection)
	// - Health improvement (SUSPECTED/DEAD -> HEALTHY): Only if remote has SIGNIFICANTLY newer info (prevent stale resurrections)
	if versionState.DigestEntry.PeerState != messages.PeerState_PEER_STATE_UNSPECIFIED {
		remoteState := versionState.DigestEntry.PeerState
		localState := peer.GetState()

		// Compare timestamps to determine who has fresher information
		remoteLastSeen := time.Unix(0, versionState.DigestEntry.LastSeenUnixNano)
		localLastSeen := peer.GetLastSeen()

		// For resurrection, we require remote info to be SIGNIFICANTLY newer (at least resurrectionThreshold)
		// This prevents flapping where nodes keep resurrecting each other based on slightly different timestamps
		resurrectionCutoff := localLastSeen.Add(s.resurrectionThreshold)
		remoteIsSignificantlyNewer := remoteLastSeen.After(resurrectionCutoff)

		// Handle state transitions based on remote state
		switch remoteState {
		case messages.PeerState_PEER_STATE_HEALTHY:
			// Only resurrect if the remote has SIGNIFICANTLY more recent information
			// This prevents stale "healthy" states from resurrecting dead peers
			if !peer.IsHealthy() && remoteIsSignificantlyNewer {
				peer.MarkHealthy()
				// Update last seen time to match remote's fresher info
				peer.SetLastSeen(remoteLastSeen)
				otelzap.L().Info("Peer resurrected based on remote info",
					zap.String("nodeID", s.GetId()),
					zap.String("peerID", peerID),
					zap.String("previousState", localState.String()),
					zap.Time("remoteLastSeen", remoteLastSeen),
					zap.Time("localLastSeen", localLastSeen),
					zap.Duration("resurrectionThreshold", s.resurrectionThreshold),
				)
			} else if !peer.IsHealthy() && !remoteIsSignificantlyNewer {
				otelzap.L().Debug("Ignoring remote healthy state (not significantly newer)",
					zap.String("nodeID", s.GetId()),
					zap.String("peerID", peerID),
					zap.Time("remoteLastSeen", remoteLastSeen),
					zap.Time("localLastSeen", localLastSeen),
					zap.Time("resurrectionCutoff", resurrectionCutoff),
					zap.Duration("resurrectionThreshold", s.resurrectionThreshold),
				)
			}
		case messages.PeerState_PEER_STATE_SUSPECTED_DEAD:
			// Always apply suspected dead state for fast failure detection
			// Even if our info is newer, if another node suspects it's dead, we should too
			if localState == messages.PeerState_PEER_STATE_HEALTHY {
				peer.MarkSuspectedDead()
				otelzap.L().Info("Peer marked suspected dead via gossip",
					zap.String("nodeID", s.GetId()),
					zap.String("peerID", peerID),
				)
			}
		case messages.PeerState_PEER_STATE_DEAD:
			// Always apply dead state for fast failure detection
			if !peer.IsDead() {
				peer.MarkDead()
				otelzap.L().Info("Peer marked dead via gossip",
					zap.String("nodeID", s.GetId()),
					zap.String("peerID", peerID),
				)
			}
		}
	}

	s.peers[peerID] = peer

	// Unmarshal and update the peer's state
	state, err := s.unmarshalAndCreateState(versionState)
	if err != nil {
		return err
	}

	currentState := s.state[peerID]

	// If we don't have a current state for this peer, just set it
	if currentState == nil {
		s.state[peerID] = state
		return nil
	}

	// Validate version monotonicity
	if currentState.GetVersion() > state.GetVersion() {
		return ErrorVersionedStateMonotonicIncreasing
	}

	if currentState.GetVersion() == state.GetVersion() && currentState.GetData() != state.GetData() {
		// conflict resolution: if the peer is "lexically smaller" than us, we just apply the peers state
		if peerID < s.GetId() {
			// however, if we do this, we also bump the state version so it's re-announced to the network
			// on the next gossip iteration
			state.SetData(state.GetData())
			s.state[peerID] = state
		}
		return ErrorVersionedStateFork
	}

	s.state[peerID] = state
	return nil
}
