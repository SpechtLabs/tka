package cluster_test

import (
	"testing"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/cluster"
	"github.com/spechtlabs/tka/pkg/cluster/mock"
	"github.com/stretchr/testify/assert"
)

type testState cluster.GossipVersionedState[string]

type versionCondition string

const (
	newerVersion   versionCondition = "newer"
	olderVersion   versionCondition = "older"
	newerWriteTime versionCondition = "newerWriteTime"
	olderWriteTime versionCondition = "olderWriteTime"
	identical      versionCondition = "identical"
)

func setupStates(t *testing.T, state testState, diff testState, diffVersion versionCondition) (testState, testState) {
	t.Helper()

	switch diffVersion {
	case newerVersion:
		// If the diff is newer, we need to apply it again to increment it's version counter
		diff.SetData(diff.GetData())

		// If the diff is older, we need to apply the state to increment it's version counter
	case olderVersion:
		state.SetData(state.GetData())

	case newerWriteTime:
		// If the diff is the same, we need to apply both
		state.SetData(state.GetData())

		// but we sleep to ensure that the diff is newer
		time.Sleep(100 * time.Millisecond)
		diff.SetData(diff.GetData())

	case olderWriteTime:
		// If the diff is the same, we need to apply both
		diff.SetData(diff.GetData())

		// but we sleep to ensure that the diff is older
		time.Sleep(100 * time.Millisecond)
		state.SetData(state.GetData())

	case identical:
		// if the diff is supposed to be identical, we should set the diff to the state
		diff = state
	}

	return state, diff
}

func TestLastWriteWinsStateApply(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name          string
		state         testState
		diff          testState
		diffVersion   versionCondition
		expectedState string
		expectedError humane.Error
	}{
		{
			name:          "diff is newer",
			state:         cluster.NewLastWriteWinsState("foo"),
			diff:          cluster.NewLastWriteWinsState("bar"),
			diffVersion:   newerVersion,
			expectedState: "bar",
			expectedError: nil,
		},
		{
			name:          "diff is older",
			state:         cluster.NewLastWriteWinsState("foo"),
			diff:          cluster.NewLastWriteWinsState("bar"),
			diffVersion:   olderVersion,
			expectedState: "foo",
			expectedError: nil,
		},
		{
			name:          "diff is newer with different diff type",
			state:         cluster.NewLastWriteWinsState("foo"),
			diff:          mock.NewMockVersionedState("bar"),
			diffVersion:   newerVersion,
			expectedState: "bar",
			expectedError: nil,
		},
		{
			name:          "diff is older with different state type",
			state:         mock.NewMockVersionedState("foo"),
			diff:          cluster.NewLastWriteWinsState("bar"),
			diffVersion:   olderVersion,
			expectedState: "foo",
			expectedError: nil,
		},
		{
			name:          "diff is equal (diff wins)",
			state:         cluster.NewLastWriteWinsState("foo"),
			diff:          cluster.NewLastWriteWinsState("bar"),
			diffVersion:   newerWriteTime,
			expectedState: "bar",
			expectedError: nil,
		},
		{
			name:          "diff is equal (state wins)",
			state:         cluster.NewLastWriteWinsState("foo"),
			diff:          cluster.NewLastWriteWinsState("bar"),
			diffVersion:   olderWriteTime,
			expectedState: "foo",
			expectedError: nil,
		},
		{
			name:          "diff is identical",
			state:         cluster.NewLastWriteWinsState("foo"),
			diff:          cluster.NewLastWriteWinsState("foo"),
			diffVersion:   identical,
			expectedState: "foo",
			expectedError: humane.New("Vector clock is out of sync. Unclear how to resolve this conflict."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, diff := setupStates(t, tt.state, tt.diff, tt.diffVersion)

			diffResult := state.Diff(diff)
			equalResult := state.Equal(diff)

			switch tt.diffVersion {
			case newerVersion:
				assert.Nil(t, diffResult)
				assert.False(t, equalResult)

			case olderVersion:
				assert.Equal(t, diffResult, state)
				assert.False(t, equalResult)

			case newerWriteTime:
				assert.Nil(t, diffResult)
				assert.False(t, equalResult)

			case olderWriteTime:
				assert.Equal(t, diffResult, state)
				assert.False(t, equalResult)

			case identical:
				assert.Equal(t, diffResult, state)
				assert.True(t, equalResult)
			}

			err := state.Apply(diff)
			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedState, state.GetData())
		})
	}
}
