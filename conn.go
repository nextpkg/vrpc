package vrpc

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/nextpkg/vrpc/internal/codec"
)

// ResponseConnection 负责将响应发送回客户端
type ResponseConnection struct {
	producer MessageProducer
	topic    string
}

// NewResponseConnection 创建一个新的响应连接
func NewResponseConnection(producer MessageProducer, topic string) *ResponseConnection {
	return &ResponseConnection{
		producer: producer,
		topic:    topic,
	}
}

// Send 发送响应到响应队列
func (r *ResponseConnection) Send(ctx context.Context, resp *codec.Response) error {
	// 序列化响应
	respBytes, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	// 发送消息到MQ
	return r.producer.Produce(ctx, fmt.Sprintf("%d", resp.Base.Id), respBytes)
}
