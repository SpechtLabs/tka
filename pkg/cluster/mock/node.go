package mock

import (
	"encoding/base64"
	"hash/fnv"
	"time"

	"github.com/spechtlabs/tka/pkg/cluster"
	"github.com/spechtlabs/tka/pkg/test"
)

func hashString(s string) string {
	hasher := fnv.New128()
	hasher.Write([]byte(s))
	return base64.StdEncoding.EncodeToString(hasher.Sum(nil))
}

type MockGossipNode struct {
	tracker  *test.CallTracker
	id       string
	addr     string
	lastSeen time.Time
}

type GossipNodeOption func(*MockGossipNode)

func WithID(id string) GossipNodeOption {
	return func(n *MockGossipNode) { n.id = id }
}

func WithNodeAddress(addr string) GossipNodeOption {
	return func(n *MockGossipNode) { n.addr = addr }
}

func WithNodeTracker(tracker *test.CallTracker) GossipNodeOption {
	return func(n *MockGossipNode) { n.tracker = tracker }
}

func WithNodeLastSeen(lastSeen time.Time) GossipNodeOption {
	return func(n *MockGossipNode) { n.lastSeen = lastSeen }
}

func NewMockGossipNode(opts ...GossipNodeOption) cluster.GossipNode {
	n := &MockGossipNode{tracker: nil}

	for _, opt := range opts {
		opt(n)
	}

	// Auto-generate ID if not provided
	if n.id == "" && n.addr != "" {
		n.id = hashString(n.addr)
	}

	return n
}

func (n *MockGossipNode) ID() string {
	if n.tracker != nil {
		n.tracker.Record("ID")
	}
	return n.id
}

func (n *MockGossipNode) String() string {
	if n.tracker != nil {
		n.tracker.Record("String")
	}
	return n.id
}

func (n *MockGossipNode) GetAddress() string {
	if n.tracker != nil {
		n.tracker.Record("GetAddress")
	}
	return n.addr
}

func (n *MockGossipNode) GetLastSeen() time.Time {
	if n.tracker != nil {
		n.tracker.Record("GetLastSeen")
	}
	return n.lastSeen
}
