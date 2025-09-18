package validator

import (
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// UnmarshalProtoTextOrBinary attempts text-format first, then binary.
func UnmarshalProtoTextOrBinary(data []byte, msg proto.Message) error {
	if err := prototext.Unmarshal(data, msg); err != nil {
		if err2 := proto.Unmarshal(data, msg); err2 != nil {
			return err2
		}
	}
	return nil
}
