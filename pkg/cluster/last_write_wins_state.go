package cluster

import (
	"github.com/sierrasoftworks/humane-errors-go"
)

// LastWriteWinsState is a implementation of the GossipVersionedState interface that uses the last write time to resolve conflicts.
type LastWriteWinsState[T SerializableAndStringable] struct {
	version Version
	data    T
}

// NewLastWriteWinsState creates a new GossipVersionedState with the given data.
// The last write time is used for resolving conflicts when the version is the same.
func NewLastWriteWinsState[T SerializableAndStringable](data T) GossipVersionedState[T] {
	return &LastWriteWinsState[T]{
		version: 0,
		data:    data,
	}
}

func (s *LastWriteWinsState[T]) Equal(other GossipVersionedState[T]) bool {
	if s.version != other.GetVersion() {
		return false
	}

	return s.data.ValuesEqual(other.GetData())
}

func (s *LastWriteWinsState[T]) GetVersion() Version {
	return s.version
}

func (s *LastWriteWinsState[T]) GetData() T {
	return s.data
}

func (s *LastWriteWinsState[T]) Copy() GossipVersionedState[T] {
	return &LastWriteWinsState[T]{
		version: s.version,
		data:    s.data,
	}
}

func (s *LastWriteWinsState[T]) SetData(data T) {
	s.version++
	s.data = data
}

func (s *LastWriteWinsState[T]) Diff(other Version) GossipVersionedState[T] {
	// If the other state is an older version, we are authorative and return a copy of ourselves
	if s.GetVersion() > other {
		return s.Copy()
	}

	// If the other state is a newer version, we return nil
	if s.GetVersion() < other {
		return nil
	}

	// If we are the same version, we return a copy of ourselves as this gives the
	// recipient the possibility to do conflict resolution of forks by using the
	// smaller node-id as the tie-breaker for a conflict.
	return s.Copy()
}

func (s *LastWriteWinsState[T]) Apply(diff GossipVersionedState[T]) humane.Error {
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

	// If versions are equal, we check if data is different
	if !s.data.ValuesEqual(diff.GetData()) {
		// This is the "last write wins" scenario - when versions are equal but data differs,
		// we apply the diff data (the incoming write wins)
		s.data = diff.GetData()
	}

	return nil
}
