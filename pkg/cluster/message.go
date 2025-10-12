package cluster

import (
	"encoding/json"
)

type GossipMessageType string

const (
	GossipMessageJoin    GossipMessageType = "join"
	GossipMessageDiff    GossipMessageType = "diff"
	GossipMessageDiffAck GossipMessageType = "diffAck"
)

type GossipMessagePayload interface {
	json.Marshaler
	json.Unmarshaler
	Equal(other GossipMessagePayload) bool
}

type GossipMessage interface {
	GetMessageType() GossipMessageType
	GetSource() GossipNode
	GetDestination() GossipNode
	GetPayload() GossipMessagePayload
}
