package cluster

import (
	"github.com/sierrasoftworks/humane-errors-go"
)

// Version is a uint64 that is used to track the version of a state.
type Version uint64

// GossipVersionedState is a vector clock implementation of the gossip state that allows for diffing and applying of states and resolving conflicts.
type GossipVersionedState[T comparable] interface {
	// Equal checks if the state is the same as another state.
	// To check if the state is equal, the version and and the data must be the same.
	Equal(other GossipVersionedState[T]) bool

	// Copy returns a copy of the state.
	Copy() GossipVersionedState[T]

	// GetVersion returns the version of the state.
	GetVersion() Version

	// GetData returns the data of the state.
	GetData() T

	// Diff returns a copy of the itselft if it is newer than the other version, otherwise (if the other version is newer than itself)it returns nil
	Diff(other Version) GossipVersionedState[T]

	// Apply applies a diff to the state.
	// If the diff is newer, it will apply the diff and return nil.
	// If the diff is older, it will return nil because we are authorative.
	// If ambiguous, it will return an error but it is up to the implementation to decide how to resolve the conflict.
	Apply(diff GossipVersionedState[T]) humane.Error

	// SetData sets the data of the state.
	// This will increment the version and set the data.
	SetData(data T)
}
