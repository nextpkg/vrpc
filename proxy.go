package vrpc

import (
	"fmt"

	"github.com/nextpkg/vrpc/internal/codec"
	"github.com/zeromicro/go-zero/core/queue"
	"google.golang.org/grpc"
)

// UpstreamProxy 将RPC请求转换为消息并发送到MQ
type UpstreamProxy struct {
	producer broker.Producer
	consumer queue.MessageQueue
}

func NewUpstreamProxy(producer MessageProducer, requestTopic string) *UpstreamProxy {
	return &UpstreamProxy{
		producer:     producer,
		requestTopic: requestTopic,
	}
}

// Send 发送请求到MQ
func (u *UpstreamProxy) Send(req *codec.Request) error {
	// 序列化请求
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	// 发送消息到MQ
	return u.producer.Produce(u.requestTopic, fmt.Sprintf("%d", req.Base.Id), reqBytes)
}

// DownstreamProxy 从MQ接收消息并转换为RPC调用
type DownstreamProxy struct {
	consumer     MessageConsumer
	responseConn *ResponseConnection
	serverConn   *grpc.ClientConn
	topic        string
}

// Start 启动下游代理
func (d *DownstreamProxy) Start(ctx context.Context) error {
	return d.consumer.Consume(ctx, d.topic, func(key string, value []byte) error {
		// 反序列化请求
		req := &Request{}
		if err := proto.Unmarshal(value, req); err != nil {
			return err
		}

		// 处理请求
		return d.handleRequest(req)
	})
}

// handleRequest 处理来自MQ的请求
func (d *DownstreamProxy) handleRequest(req *Request) error {
	// 创建方法调用
	methodDesc := d.getMethodDesc(req.Method)
	if methodDesc == nil {
		return fmt.Errorf("method not found: %s", req.Method)
	}

	// 创建请求消息
	reqMsg := dynamic.NewMessage(methodDesc.GetInputType())
	if err := reqMsg.Unmarshal(req.Payload); err != nil {
		return err
	}

	// 创建响应消息
	respMsg := dynamic.NewMessage(methodDesc.GetOutputType())

	// 执行调用
	err := d.serverConn.Invoke(context.Background(), req.Method, reqMsg, respMsg)

	// 构建响应
	response := &Response{
		CorrelationID: req.CorrelationID,
	}

	if err != nil {
		response.Error = err.Error()
	} else {
		respBytes, err := proto.Marshal(respMsg)
		if err != nil {
			response.Error = err.Error()
		} else {
			response.Payload = respBytes
		}
	}

	// 发送响应到客户端
	return d.responseConn.Send(response)
}

// MessageConsumer 消息消费者接口
type MessageConsumer interface {
	Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error
}
