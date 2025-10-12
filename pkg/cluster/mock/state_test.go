package mock

import (
	"testing"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/pkg/test"
	"github.com/stretchr/testify/assert"
)

type testState string

type versionCondition string

const (
	newerVersion versionCondition = "newer"
	olderVersion versionCondition = "older"
	identical    versionCondition = "identical"
)

func TestLastWriteWinsStateApply(t *testing.T) {
	t.Helper()
	t.Parallel()

	tests := []struct {
		name          string
		state         testState
		diff          testState
		diffVersion   versionCondition
		expectedState testState
		expectedError humane.Error
	}{
		{
			name:          "diff is newer",
			state:         testState("foo"),
			diff:          testState("bar"),
			diffVersion:   newerVersion,
			expectedState: testState("bar"),
			expectedError: nil,
		},
		{
			name:          "diff is older",
			state:         testState("foo"),
			diff:          testState("bar"),
			diffVersion:   olderVersion,
			expectedState: testState("foo"),
			expectedError: nil,
		},
		{
			name:          "diff is identical",
			state:         testState("foo"),
			diff:          testState("foo"),
			diffVersion:   identical,
			expectedState: testState("foo"),
			expectedError: humane.New("Vector clock is out of sync. Unclear how to resolve this conflict."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			stateTracker := test.NewCallTracker()
			diffTracker := test.NewCallTracker()

			state := NewMockVersionedState(tt.state, WithMockVersionedStateTracker[testState](stateTracker))
			diff := NewMockVersionedState(tt.diff, WithMockVersionedStateTracker[testState](diffTracker))

			switch tt.diffVersion {
			case newerVersion:
				// If the diff is newer, we need to apply it again to increment it's version counter
				diff.SetData(diff.GetData())
				diffTracker.CalledOnce("SetData")

				// If the diff is older, we need to apply the state to increment it's version counter
			case olderVersion:
				state.SetData(state.GetData())
				stateTracker.CalledOnce("SetData")

			case identical:
				// if the diff is supposed to be identical, we should set the diff to the state
				diff = state
			}

			diffResult := state.Diff(diff)
			stateTracker.CalledOnce("Diff")

			equalResult := state.Equal(diff)
			stateTracker.CalledOnce("Equal")

			switch tt.diffVersion {
			case newerVersion:
				assert.Nil(t, diffResult)
				assert.False(t, equalResult)

			case olderVersion:
				assert.Equal(t, diffResult, state.Copy())
				assert.False(t, equalResult)

			case identical:
				assert.Equal(t, diffResult, state.Copy())
				assert.True(t, equalResult)
			}

			err := state.Apply(diff)
			stateTracker.CalledOnce("Apply")

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedState, state.GetData())
		})
	}
}
