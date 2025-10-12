package mock

import (
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/cluster"
	"github.com/spechtlabs/tka/pkg/test"
)

type MockVersionedState[T comparable] struct {
	version cluster.Version
	data    T
	tracker *test.CallTracker
}

type GossipVersionedStateOption[T comparable] func(*MockVersionedState[T])

func WithMockVersionedStateTracker[T comparable](tracker *test.CallTracker) GossipVersionedStateOption[T] {
	return func(s *MockVersionedState[T]) { s.tracker = tracker }
}

// NewMockVersionedState creates a new GossipVersionedState with the given data.
func NewMockVersionedState[T comparable](data T, opts ...GossipVersionedStateOption[T]) cluster.GossipVersionedState[T] {
	s := &MockVersionedState[T]{
		version: 0,
		data:    data,
		tracker: nil,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *MockVersionedState[T]) Equal(other cluster.GossipVersionedState[T]) bool {
	if s.version != other.GetVersion() {
		return false
	}
	return s.data == other.GetData()
}

func (s *MockVersionedState[T]) GetVersion() cluster.Version {
	if s.tracker != nil {
		s.tracker.Record("GetVersion")
	}
	return s.version
}

func (s *MockVersionedState[T]) GetData() T {
	if s.tracker != nil {
		s.tracker.Record("GetData")
	}
	return s.data
}

func (s *MockVersionedState[T]) Copy() cluster.GossipVersionedState[T] {
	if s.tracker != nil {
		s.tracker.Record("Copy")
	}
	return &MockVersionedState[T]{
		version: s.version,
		data:    s.data,
	}
}

func (s *MockVersionedState[T]) SetData(data T) {
	if s.tracker != nil {
		s.tracker.Record("SetData")
	}
	s.version++
	s.data = data
}

func (s *MockVersionedState[T]) Diff(other cluster.GossipVersionedState[T]) cluster.GossipVersionedState[T] {
	if s.tracker != nil {
		s.tracker.Record("Diff")
	}
	// If the other state is an older version, we are authorative and return a copy of ourselves
	if s.GetVersion() > other.GetVersion() {
		return s.Copy()
	}

	// If the other state is a newer version, we return nil
	if s.GetVersion() < other.GetVersion() {
		return nil
	}

	// in all other cases, we return a copy of ourselves
	return s.Copy()
}

func (s *MockVersionedState[T]) Apply(diff cluster.GossipVersionedState[T]) humane.Error {
	if s.tracker != nil {
		s.tracker.Record("Apply")
	}
	// If the diff is an older version, we are authorative and don't need to apply it
	if s.version > diff.GetVersion() {
		return nil
	}

	// If the diff is a newer version, we need to apply it
	if s.version < diff.GetVersion() {
		s.version = diff.GetVersion()
		s.data = diff.GetData()
		return nil
	}

	// Unclear how we got here, but we should return an error
	return humane.New("Vector clock is out of sync. Unclear how to resolve this conflict.")
}
