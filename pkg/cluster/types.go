package cluster

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// SerializableString is a string type that implements SerializableAndStringable interface.
// It provides msgpack serialization and string representation for use in gossip states.
type SerializableString string

// MarshalMsgpack marshals the string to msgpack format.
func (s SerializableString) MarshalMsgpack() ([]byte, error) {
	return msgpack.Marshal(string(s))
}

// String implements fmt.Stringer interface.
func (s SerializableString) String() string {
	return string(s)
}

// UnmarshalMsgpack unmarshals a SerializableString from msgpack format.
// Note: This method is required by msgpack.Unmarshaler interface, but msgpack
// will actually call it on a pointer receiver. The value receiver here is only
// for interface satisfaction purposes.
func (s SerializableString) UnmarshalMsgpack(b []byte) error {
	// This method is not used directly by msgpack for value types.
	// msgpack will automatically convert to pointer and call the pointer version if needed.
	// However, since we're using UnmarshalSerializableString helper, this should not be called.
	return fmt.Errorf("UnmarshalMsgpack should not be called on value receiver - use UnmarshalSerializableString instead")
}

func UnmarshalSerializableString(b []byte) (SerializableString, error) {
	var str string
	if err := msgpack.Unmarshal(b, &str); err != nil {
		return "", fmt.Errorf("failed to unmarshal SerializableString: %w", err)
	}
	return SerializableString(str), nil
}
