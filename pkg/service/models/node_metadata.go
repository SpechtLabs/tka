package models

import (
	"encoding/json"
	"reflect"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/vmihailenco/msgpack/v5"
)

// NodeMetadata contains the information to be gossiped about a TKA server node.
// This metadata is shared via the gossip protocol and allows clients to discover
// all available clusters and their connection details.
type NodeMetadata struct {
	// APIEndpoint is the Kubernetes API server URL for this cluster
	APIEndpoint string `json:"apiEndpoint" msgpack:"apiEndpoint" example:"https://api.prod-us.example.com:6443"`

	// APIPort is the port of the TKA API server
	APIPort int `json:"apiPort" msgpack:"apiPort" example:"443"`

	// Labels are key-value pairs used to identify and categorize the cluster
	Labels map[string]string `json:"labels" msgpack:"labels" example:"environment:production,region:us-west-2"`
}

// Marshal marshals the NodeMetadata to msgpack format.
func (n NodeMetadata) Marshal() ([]byte, humane.Error) {
	data, err := msgpack.Marshal(n)
	if err != nil {
		return nil, humane.Wrap(err, "failed to marshal NodeMetadata")
	}
	return data, nil
}

// Unmarshal unmarshals the NodeMetadata from msgpack format.
func (n NodeMetadata) Unmarshal(data []byte, v interface{}) humane.Error {
	var metadata NodeMetadata
	if err := msgpack.Unmarshal(data, &metadata); err != nil {
		return humane.Wrap(err, "failed to unmarshal NodeMetadata")
	}
	*v.(*NodeMetadata) = metadata
	return nil
}

// String implements fmt.Stringer interface.
func (n NodeMetadata) String() string {
	// Use JSON for string representation as it is human readable
	data, _ := json.Marshal(n)
	return string(data)
}

// ValuesEqual checks if the values are equal.
func (n NodeMetadata) ValuesEqual(other interface{}) bool {
	o, ok := other.(NodeMetadata)
	if !ok {
		return false
	}

	if n.APIEndpoint != o.APIEndpoint {
		return false
	}
	if n.APIPort != o.APIPort {
		return false
	}

	return reflect.DeepEqual(n.Labels, o.Labels)
}
