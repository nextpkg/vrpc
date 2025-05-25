package rpcbus

import "context"

// Producer 消息生产者接口
type Producer interface {
	Produce(ctx context.Context, key string, value []byte) error
	Close() error
}
