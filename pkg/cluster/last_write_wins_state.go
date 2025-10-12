package cluster

import (
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
)

// LastWriteWinsState is a implementation of the GossipVersionedState interface that uses the last write time to resolve conflicts.
type LastWriteWinsState[T comparable] struct {
	version   Version
	lastWrite time.Time
	data      T
}

// NewLastWriteWinsState creates a new GossipVersionedState with the given data.
// The last write time is used for resolving conflicts when the version is the same.
func NewLastWriteWinsState[T comparable](data T) GossipVersionedState[T] {
	return &LastWriteWinsState[T]{
		version:   0,
		lastWrite: time.Now(),
		data:      data,
	}
}

func (s *LastWriteWinsState[T]) Equal(other GossipVersionedState[T]) bool {
	if s.version != other.GetVersion() {
		return false
	}

	if diffLww, ok := other.(*LastWriteWinsState[T]); ok && !diffLww.lastWrite.Equal(s.lastWrite) {
		return false
	}

	return s.data == other.GetData()
}

func (s *LastWriteWinsState[T]) GetVersion() Version {
	return s.version
}

func (s *LastWriteWinsState[T]) GetData() T {
	return s.data
}

func (s *LastWriteWinsState[T]) Copy() GossipVersionedState[T] {
	return &LastWriteWinsState[T]{
		version:   s.version,
		lastWrite: s.lastWrite,
		data:      s.data,
	}
}

func (s *LastWriteWinsState[T]) SetData(data T) {
	s.version++
	s.lastWrite = time.Now()
	s.data = data
}

func (s *LastWriteWinsState[T]) Diff(other GossipVersionedState[T]) GossipVersionedState[T] {
	// If the other state is an older version, we are authorative and return a copy of ourselves
	if s.GetVersion() > other.GetVersion() {
		return s.Copy()
	}

	// If the other state is a newer version, we return nil
	if s.GetVersion() < other.GetVersion() {
		return nil
	}

	// If the other state is the same version, we must look deeper at the last write time
	if diffLww, ok := other.(*LastWriteWinsState[T]); ok {
		// If we are newer, we are authorative and return a copy of ourselves
		if s.lastWrite.After(diffLww.lastWrite) {
			return s.Copy()
		}

		// If we are older, we are not authorative and return nil
		if s.lastWrite.Before(diffLww.lastWrite) {
			return nil
		}
	}

	// In doubt: return a copy of ourselves
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
		if diffLww, ok := diff.(*LastWriteWinsState[T]); ok {
			s.lastWrite = diffLww.lastWrite
		} else {
			s.lastWrite = time.Now()
		}
		return nil
	}

	// If the diff is the same version, we use the last write time to resolve the conflict
	if diffLww, ok := diff.(*LastWriteWinsState[T]); ok {
		// If we are newer, nothing needs to be done
		if s.lastWrite.After(diffLww.lastWrite) {
			return nil
		}

		// If we are older, we need to apply the diff
		if s.lastWrite.Before(diffLww.lastWrite) {
			s.version = diffLww.GetVersion()
			s.data = diffLww.GetData()
			return nil
		}
	}

	// Unclear how we got here, but we should return an error
	return humane.New("Vector clock is out of sync. Unclear how to resolve this conflict.")
}
