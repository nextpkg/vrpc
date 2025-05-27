// pkg/mq/kafka.go
package rpcbus

import (
	"context"

	"github.com/nextpkg/vrpc/internal/proto/gen"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/proto"
)

// KafkaPublisher Kafka消息发布者
type KafkaPublisher struct {
	pusher *kq.Pusher
}

// KafkaConsumer Kafka消息消费者
type KafkaConsumer struct {
	queue  interface{}
	config kq.KqConf
}

// NewKafkaPublisher 创建Kafka发布者
func NewKafkaPublisher(config *Config) (*KafkaPublisher, error) {
	// 从settings中获取topic，如果没有则使用默认值
	topic := "vrpc-requests"
	if config.Settings != nil {
		if t, ok := config.Settings["topic"].(string); ok {
			topic = t
		}
	}
	pusher := kq.NewPusher(config.Brokers, topic)
	return &KafkaPublisher{pusher: pusher}, nil
}

func (p *KafkaPublisher) Publish(ctx context.Context, topic string, message *gen.VRPCRequest) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return p.pusher.Push(ctx, string(data))
}

func (p *KafkaPublisher) PublishResponse(ctx context.Context, topic string, message *gen.VRPCResponse) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return p.pusher.Push(ctx, string(data))
}

func (p *KafkaPublisher) Close() error {
	return p.pusher.Close()
}

// NewKafkaConsumerWithRequestHandler 创建处理请求的Kafka消费者
func NewKafkaConsumerWithRequestHandler(config *Config, consumerGroup string, topic string, handler RequestHandler) (*KafkaConsumer, error) {
	consumerConfig := kq.KqConf{
		Brokers:    config.Brokers,
		Group:      consumerGroup,
		Topic:      topic,
		Processors: 8, // 默认处理器数量
		Offset:     "last",
	}

	// 应用自定义配置
	if settings := config.Settings; settings != nil {
		if processors, ok := settings["processors"].(int); ok {
			consumerConfig.Processors = processors
		}
		if offset, ok := settings["offset"].(string); ok {
			consumerConfig.Offset = offset
		}
	}

	// 创建队列并设置请求处理函数
	queue := kq.MustNewQueue(consumerConfig, kq.WithHandle(func(ctx context.Context, key, value string) error {
		var req gen.VRPCRequest
		if err := proto.Unmarshal([]byte(value), &req); err != nil {
			logx.Errorf("Failed to unmarshal request: %v", err)
			return err
		}
		return handler(ctx, &req)
	}))

	return &KafkaConsumer{queue: queue, config: consumerConfig}, nil
}

// NewKafkaConsumerWithResponseHandler 创建处理响应的Kafka消费者
func NewKafkaConsumerWithResponseHandler(config *Config, consumerGroup string, topic string, handler ResponseHandler) (*KafkaConsumer, error) {
	consumerConfig := kq.KqConf{
		Brokers:    config.Brokers,
		Group:      consumerGroup,
		Topic:      topic,
		Processors: 8, // 默认处理器数量
		Offset:     "last",
	}

	// 应用自定义配置
	if settings := config.Settings; settings != nil {
		if processors, ok := settings["processors"].(int); ok {
			consumerConfig.Processors = processors
		}
		if offset, ok := settings["offset"].(string); ok {
			consumerConfig.Offset = offset
		}
	}

	// 创建队列并设置响应处理函数
	queue := kq.MustNewQueue(consumerConfig, kq.WithHandle(func(ctx context.Context, key, value string) error {
		var resp gen.VRPCResponse
		if err := proto.Unmarshal([]byte(value), &resp); err != nil {
			logx.Errorf("Failed to unmarshal response: %v", err)
			return err
		}
		return handler(ctx, &resp)
	}))

	return &KafkaConsumer{queue: queue, config: consumerConfig}, nil
}

// NewKafkaConsumer 创建Kafka消费者（兼容性方法）
func NewKafkaConsumer(config *Config, consumerGroup string, topic string) (*KafkaConsumer, error) {
	consumerConfig := kq.KqConf{
		Brokers:    config.Brokers,
		Group:      consumerGroup,
		Topic:      topic,
		Processors: 8, // 默认处理器数量
		Offset:     "last",
	}

	// 应用自定义配置
	if settings := config.Settings; settings != nil {
		if processors, ok := settings["processors"].(int); ok {
			consumerConfig.Processors = processors
		}
		if offset, ok := settings["offset"].(string); ok {
			consumerConfig.Offset = offset
		}
	}

	// 创建队列，使用空的处理函数
	queue := kq.MustNewQueue(consumerConfig, kq.WithHandle(func(ctx context.Context, key, value string) error {
		// 空处理函数，需要通过其他方式设置实际处理逻辑
		return nil
	}))

	return &KafkaConsumer{queue: queue, config: consumerConfig}, nil
}

func (c *KafkaConsumer) Consume(ctx context.Context, topic string, handler RequestHandler) error {
	// 启动队列
	if queue, ok := c.queue.(interface{ Start() }); ok {
		go func() {
			queue.Start()
		}()
	}
	return nil
}

func (c *KafkaConsumer) ConsumeResponse(ctx context.Context, topic string, handler ResponseHandler) error {
	// 启动队列
	if queue, ok := c.queue.(interface{ Start() }); ok {
		go func() {
			queue.Start()
		}()
	}
	return nil
}

func (c *KafkaConsumer) Close() error {
	if queue, ok := c.queue.(interface{ Stop() error }); ok {
		return queue.Stop()
	}
	return nil
}
