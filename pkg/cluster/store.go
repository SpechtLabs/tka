package cluster

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
)

type GossipDigest map[string]*messages.DigestEntry

func NewDigestEntry(version uint64, peer *GossipNode) (*messages.DigestEntry, humane.Error) {
	if peer == nil {
		return nil, humane.New("peer is nil", "pass in a valid peer")
	}

	return &messages.DigestEntry{
		Version:          version,
		Address:          peer.GetAddress(),
		LastSeenUnixNano: peer.GetLastSeen().UnixNano(),
		PeerState:        peer.GetState(),
	}, nil
}

func NewDigestEntryFromPeerDigest(version uint64, peer *messages.DigestEntry) (*messages.DigestEntry, humane.Error) {
	if peer == nil {
		return nil, humane.New("peer is nil", "pass in a valid peer")
	}

	return &messages.DigestEntry{
		Version:          version,
		Address:          peer.Address,
		LastSeenUnixNano: peer.LastSeenUnixNano,
		PeerState:        peer.PeerState,
	}, nil
}

type GossipDiff map[string]*messages.GossipVersionedState

func (d GossipDiff) ToString() string {
	lines := make([]string, 0, len(d))
	for peerId, versionState := range d {
		lines = append(lines, fmt.Sprintf("%s: %s", peerId, versionState.GetData()))
	}
	return strings.Join(lines, "\n")
}

type GossipStore[T SerializableAndStringable] interface {
	// GetId returns the unique id of the local node
	GetId() string

	// GetPeers returns all the gossip nodes in the cluster
	GetPeers() []GossipNode

	// GetPeer returns a specific peer by id
	GetPeer(peerID string) *GossipNode

	// Heartbeat updates the last seen time of the node
	Heartbeat(peerId string, address string)

	// IncrementPeerFailure increments the consecutive failure count for a peer
	IncrementPeerFailure(peerID string, threshold int)

	// MarkPeerSuspectedDead marks a peer as suspected dead
	MarkPeerSuspectedDead(peerID string)

	// MarkPeerDead marks a peer as dead
	MarkPeerDead(peerID string)

	// ResurrectPeer marks a peer as healthy (used when a suspected dead peer responds)
	ResurrectPeer(peerID string)

	// RemoveStalePeers removes peers that have exceeded the staleness threshold
	RemoveStalePeers(threshold int) []string

	// Digest returns the version map of the local node (all connected peer nodes and their state versions)
	Digest() (GossipDigest, []humane.Error)

	// Diff returns the difference between the local node's digest and another digest
	Diff(other GossipDigest) (GossipDiff, []humane.Error)

	// Apply applies a diff to the local node's state
	Apply(diff GossipDiff) []humane.Error

	// ProcessDigestForPeerStates processes peer state information from a remote digest.
	// This updates peer states based on what other nodes report about peers in the cluster.
	// This is separate from Apply because digests may contain peer state information for
	// peers that aren't included in the state delta.
	ProcessDigestForPeerStates(remoteDigest GossipDigest) []humane.Error

	// SetData sets the status of the local node
	SetData(data T)

	// GetDisplayData returns the display data for the local node and all connected peer nodes
	GetDisplayData() []NodeDisplayData
}

type NodeDisplayData struct {
	ID          string
	Address     string
	LastSeen    time.Time
	Version     Version
	State       string
	LastUpdated time.Time
	IsLocal     bool
	PeerState   messages.PeerState
}

func hashString(s string) string {
	hasher := fnv.New128()
	hasher.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

func shaHashString(s string) string {
	hasher := sha1.New()
	hasher.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
