package cluster

import (
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/vmihailenco/msgpack/v5"
)

// SerializableString is a string type that implements SerializableAndStringable interface.
// It provides msgpack serialization and string representation for use in gossip states.
type SerializableString string

// MarshalMsgpack marshals the string to msgpack format.
func (s SerializableString) Marshal() ([]byte, humane.Error) {
	data, err := msgpack.Marshal(string(s))
	if err != nil {
		return nil, humane.Wrap(err, "failed to marshal SerializableString")
	}
	return data, nil
}

func (s SerializableString) Unmarshal(data []byte, v interface{}) humane.Error {
	var str string
	if err := msgpack.Unmarshal(data, &str); err != nil {
		return humane.Wrap(err, "failed to unmarshal SerializableString")
	}
	*v.(*SerializableString) = SerializableString(str)
	return nil
}

// String implements fmt.Stringer interface.
func (s SerializableString) String() string {
	return string(s)
}
