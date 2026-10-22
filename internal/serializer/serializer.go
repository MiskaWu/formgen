package serializer

import "google.golang.org/protobuf/reflect/protoreflect"

// ProtoSerializer 負責將數據序列化為 protobuf 格式
type ProtoSerializer struct {
	protoFilePath string
	messageCache  map[string]protoreflect.MessageDescriptor
}

// NewProtoSerializer 創建新的序列化器
func NewProtoSerializer(protoFilePath string) (*ProtoSerializer, error) {
	return &ProtoSerializer{
		protoFilePath: protoFilePath,
		messageCache:  make(map[string]protoreflect.MessageDescriptor),
	}, nil
}
