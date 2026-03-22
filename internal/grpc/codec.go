package grpc

import (
	"fmt"

	"google.golang.org/grpc/encoding"
	"google.golang.org/protobuf/proto"
)

// InstallCodec replaces the global gRPC "proto" codec with the raw-bytes
// passthrough implementation. Must be called before the gRPC server starts.
// It is called explicitly from NewServer so it only affects processes that
// actually start a gRPC server (HTTP-only runs are unaffected).
func InstallCodec() {
	encoding.RegisterCodec(rawCodec{})
}

// rawCodec is a gRPC codec that passes *[]byte values through unchanged and
// falls back to standard proto marshalling for all other types (e.g. gRPC
// reflection service messages).
type rawCodec struct{}

func (rawCodec) Name() string { return "proto" }

// Marshal encodes v into wire bytes.
// *[]byte → returned as-is (raw proto frame passthrough).
// proto.Message → standard protobuf encoding.
func (rawCodec) Marshal(v interface{}) ([]byte, error) {
	switch m := v.(type) {
	case *[]byte:
		if m == nil {
			return nil, nil
		}
		return *m, nil
	case []byte:
		return m, nil
	case proto.Message:
		b, err := proto.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("rawCodec marshal: %w", err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("rawCodec: cannot marshal type %T", v)
	}
}

// Unmarshal decodes wire bytes into v.
// *[]byte → bytes copied in directly (raw passthrough).
// proto.Message → standard protobuf decoding.
func (rawCodec) Unmarshal(data []byte, v interface{}) error {
	switch m := v.(type) {
	case *[]byte:
		if m == nil {
			return fmt.Errorf("rawCodec: nil *[]byte destination")
		}
		*m = make([]byte, len(data))
		copy(*m, data)
		return nil
	case proto.Message:
		return proto.Unmarshal(data, m)
	default:
		return fmt.Errorf("rawCodec: cannot unmarshal into type %T", v)
	}
}
