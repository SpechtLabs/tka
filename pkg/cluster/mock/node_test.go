package mock

import (
	"testing"
	"time"

	"github.com/spechtlabs/tka/pkg/test"
	"github.com/stretchr/testify/assert"
)

func TestMockGossipNode(t *testing.T) {
	lastSeen := time.Now()

	tests := []struct {
		name         string
		nodeId       string
		expectedId   string
		nodeAddr     string
		expectedAddr string
		lastSeen     time.Time
	}{
		{
			name:         "with id and address",
			nodeId:       "foo",
			expectedId:   "foo",
			nodeAddr:     "127.0.0.1:8123",
			expectedAddr: "127.0.0.1:8123",
		},
		{
			name:         "with address no id",
			nodeAddr:     "127.0.0.1:8123",
			expectedAddr: "127.0.0.1:8123",
			expectedId:   "6byxK6SAuKc/QtSd11fPug==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := test.NewCallTracker()
			node := NewMockGossipNode(
				WithID(tt.nodeId),
				WithNodeAddress(tt.nodeAddr),
				WithNodeTracker(tracker),
				WithNodeLastSeen(lastSeen),
			)

			// Verify values
			assert.Equal(t, tt.expectedId, node.ID())
			assert.Equal(t, tt.expectedAddr, node.GetAddress())
			assert.Equal(t, tt.expectedId, node.String())
			assert.Equal(t, lastSeen, node.GetLastSeen())

			// Verify calls
			assert.True(t, tracker.CalledOnce("ID"))
			assert.True(t, tracker.CalledAtLeast("GetAddress", 1))
			assert.Equal(t, 1, tracker.Called("ID"))
		})
	}
}
