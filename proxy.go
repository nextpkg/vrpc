package vrpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/nextpkg/vrpc/internal/codec"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// MessageProducer 消息生产者接口
type MessageProducer interface {
	Produce(ctx context.Context, key string, value []byte) error
	Close() error
}

// MessageConsumer 消息消费者接口
type MessageConsumer interface {
	Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error
}

// UpstreamProxy 将RPC请求转换为消息并发送到MQ
type UpstreamProxy struct {
	producer     MessageProducer
	requestTopic string
}

// NewUpstreamProxy 创建一个新的上游代理
func NewUpstreamProxy(producer MessageProducer, requestTopic string) *UpstreamProxy {
	return &UpstreamProxy{
		producer:     producer,
		requestTopic: requestTopic,
	}
}

// Send 发送请求到MQ
func (u *UpstreamProxy) Send(ctx context.Context, req *codec.Request) error {
	// 序列化请求
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	// 发送消息到MQ
	return u.producer.Produce(ctx, fmt.Sprintf("%d", req.Base.Id), reqBytes)
}

// DownstreamProxy 从MQ接收消息并转换为RPC调用
type DownstreamProxy struct {
	consumer     MessageConsumer
	responseConn *ResponseConnection
	serverConn   *grpc.ClientConn
	topic        string
	fileDescSet  *protoregistry.Files
}

// NewDownstreamProxy 创建一个新的下游代理
func NewDownstreamProxy(consumer MessageConsumer, responseConn *ResponseConnection, serverConn *grpc.ClientConn, topic string) *DownstreamProxy {
	return &DownstreamProxy{
		consumer:     consumer,
		responseConn: responseConn,
		serverConn:   serverConn,
		topic:        topic,
		fileDescSet:  protoregistry.GlobalFiles,
	}
}

// Start 启动下游代理
func (d *DownstreamProxy) Start(ctx context.Context) error {
	return d.consumer.Consume(ctx, d.topic, func(key string, value []byte) error {
		// 反序列化请求
		req := &codec.Request{}
		if err := proto.Unmarshal(value, req); err != nil {
			return err
		}

		// 处理请求
		return d.handleRequest(ctx, req)
	})
}

// handleRequest 处理来自MQ的请求
func (d *DownstreamProxy) handleRequest(ctx context.Context, req *codec.Request) error {
	if req.Base == nil {
		return errors.New("request base is nil")
	}

	// 构建完整方法名
	methodName := req.Base.Method
	if req.Base.Service != "" {
		methodName = fmt.Sprintf("/%s/%s", req.Base.Service, req.Base.Method)
	}

	// 执行调用
	respBytes, err := d.invokeMethod(ctx, methodName, req.Base.Payload)

	// 构建响应
	resp := &codec.Response{
		Base: &codec.Base{
			Id:        req.Base.Id,
			Service:   req.Base.Service,
			Method:    req.Base.Method,
			Timestamp: req.Base.Timestamp,
		},
	}

	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Base.Payload = respBytes
	}

	// 发送响应到客户端
	return d.responseConn.Send(ctx, resp)
}

// invokeMethod 调用gRPC方法
func (d *DownstreamProxy) invokeMethod(ctx context.Context, method string, payload []byte) ([]byte, error) {
	// 这里简化处理，直接使用原始字节作为请求和响应
	var respBytes []byte
	err := d.serverConn.Invoke(ctx, method, payload, &respBytes)
	return respBytes, err
}
