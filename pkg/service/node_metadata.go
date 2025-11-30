package service

import (
	"encoding/json"
	"reflect"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/vmihailenco/msgpack/v5"
)

// NodeMetadata contains the information to be gossiped about a TKA server node.
type NodeMetadata struct {
	APIEndpoint string            `json:"apiEndpoint" msgpack:"apiEndpoint"`
	APIPort     int               `json:"apiPort" msgpack:"apiPort"`
	Labels      map[string]string `json:"labels" msgpack:"labels"`
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
