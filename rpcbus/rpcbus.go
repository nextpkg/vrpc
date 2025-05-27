package rpcbus

import (
	"context"

	"github.com/nextpkg/vrpc/internal/proto/gen"
)

// Publisher 消息发布者接口
type Publisher interface {
	Publish(ctx context.Context, topic string, message *gen.VRPCRequest) error
	PublishResponse(ctx context.Context, topic string, message *gen.VRPCResponse) error
	Close() error
}

// Consumer 消息消费者接口
type Consumer interface {
	Consume(ctx context.Context, topic string, handler RequestHandler) error
	ConsumeResponse(ctx context.Context, topic string, handler ResponseHandler) error
	Close() error
}

// RequestHandler 请求处理函数
type RequestHandler func(ctx context.Context, message *gen.VRPCRequest) error

// ResponseHandler 响应处理函数
type ResponseHandler func(ctx context.Context, message *gen.VRPCResponse) error

// Config MQ配置
type Config struct {
	Type     string         `json:"type" yaml:"type"`
	Brokers  []string       `json:"brokers" yaml:"brokers"`
	Settings map[string]any `json:"settings" yaml:"settings"`
}
