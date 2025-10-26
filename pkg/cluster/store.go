package cluster

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/cluster/messages"
)

type GossipDigest map[string]*messages.DigestEntry

type GossipDiff map[string]*messages.GossipVersionedState

func (d GossipDiff) ToString() string {
	lines := make([]string, 0, len(d))
	for peerId, versionState := range d {
		lines = append(lines, fmt.Sprintf("%s: %s", peerId, versionState.GetData()))
	}
	return strings.Join(lines, "\n")
}

type GossipStore interface {
	// GetId returns the unique id of the local node
	GetId() string

	// GetPeers returns all the gossip nodes in the cluster
	GetPeers() []GossipNode

	// Digest returns the version map of the local node (all connected peer nodes and their state versions)
	Digest() GossipDigest

	// Heartbeat updates the last seen time of the node
	Heartbeat(peerId string, address string) humane.Error

	// Diff returns the difference between the local node's digest and another digest
	Diff(other GossipDigest) GossipDiff

	// Apply applies a diff to the local node's state
	Apply(diff GossipDiff) humane.Error

	// SetData sets the status of the local node
	SetData(data string) humane.Error

	// GetDisplayData returns the display data for the local node and all connected peer nodes
	GetDisplayData() []NodeDisplayData
}

type GossipNodeState struct {
	node        GossipNode
	state       GossipVersionedState[string]
	lastUpdated time.Time
}

func (s *GossipNodeState) SetData(data string) {
	s.node.lastSeen = time.Now()
	s.state.SetData(data)
	s.lastUpdated = time.Now()
}

func (s *GossipNodeState) GetData() string {
	return s.state.GetData()
}

func (s *GossipNodeState) GetVersion() Version {
	return s.state.GetVersion()
}

type NodeDisplayData struct {
	ID          string
	Address     string
	LastSeen    time.Time
	Version     Version
	State       string
	LastUpdated time.Time
	IsLocal     bool
}

func hashString(s string) string {
	hasher := fnv.New128()
	hasher.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}
