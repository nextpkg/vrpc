// pkg/codec/codec.go
package codec

import (
	"encoding/json"

	"google.golang.org/protobuf/proto"
)

// Serialize 序列化请求
func Serialize(req interface{}) ([]byte, error) {
	// 优先使用protobuf
	if protoMsg, ok := req.(proto.Message); ok {
		return proto.Marshal(protoMsg)
	}

	// fallback到JSON
	return json.Marshal(req)
}

// Deserialize 反序列化响应
func Deserialize(data []byte, reply interface{}) error {
	// 优先使用protobuf
	if protoMsg, ok := reply.(proto.Message); ok {
		return proto.Unmarshal(data, protoMsg)
	}

	// fallback到JSON
	return json.Unmarshal(data, reply)
}
