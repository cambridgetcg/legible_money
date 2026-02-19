package app

// proto_compat.go bridges protoc-gen-go v2 types to the gogoproto registry.
//
// Our proto files are generated with protoc-gen-go (Google protobuf v2 API) which
// registers message types with google.golang.org/protobuf's global registry. However,
// the Cosmos SDK v0.50's unknownproto field checker (codec/unknownproto/unknown_fields.go)
// uses gogoproto.MessageType() to resolve nested message types during tx decoding.
// gogoproto.MessageType() only checks its own registry (populated by proto.RegisterType).
//
// Without this bridge, any tx containing a nested message type (e.g., MsgCreateToken with
// TokenFeatures, MsgRegisterValidator with ValidatorDescription) fails with:
//   "failed to retrieve the message of type "legible.*.v1.SomeType": tx parse error"
//
// This init() runs after all imported packages' init() functions, so all protoc-gen-go
// types are already registered with the v2 global registry by the time we iterate.

import (
	"strings"

	gogoproto "github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func init() {
	registerLegibleProtoTypes()
}

// registerLegibleProtoTypes iterates all message types in the google protobuf v2
// global registry and registers any legible.* types with the gogoproto registry.
func registerLegibleProtoTypes() {
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		fullName := string(mt.Descriptor().FullName())
		if !strings.HasPrefix(fullName, "legible.") {
			return true
		}
		// Skip if already registered with gogoproto
		if gogoproto.MessageType(fullName) != nil {
			return true
		}
		// Get the underlying Go type (e.g., *TokenFeatures) from the v2 message type.
		// protoc-gen-go types implement both v2 (ProtoReflect) and v1 (Reset/String/ProtoMessage)
		// interfaces, so the type assertion to gogoproto.Message succeeds.
		goType := mt.Zero().Interface()
		if gogoMsg, ok := goType.(gogoproto.Message); ok {
			gogoproto.RegisterType(gogoMsg, fullName)
		}
		return true
	})
}
