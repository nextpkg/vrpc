package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-zero/core/logx"
)

// KafkaConsumer 实现了MessageConsumer接口，用于从Kafka消费消息
type KafkaConsumer struct {
	cfg      *rpcbus.KafkaConfig
	consumer sarama.ConsumerGroup
	handlers map[string]rpcbus.MessageHandler
	mu       sync.Mutex
	wg       sync.WaitGroup
	cancel   context.CancelFunc
}

// NewKafkaConsumer 创建一个新的Kafka消费者
func NewKafkaConsumer(cfg *rpcbus.KafkaConfig) *KafkaConsumer {
	return &KafkaConsumer{
		cfg:      cfg,
		handlers: make(map[string]rpcbus.MessageHandler),
	}
}

// Consume 从指定的主题消费消息
func (c *KafkaConsumer) Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error {
	// 创建Kafka配置
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Version = sarama.V2_0_0_0

	// 创建消费者组
	consumerGroup, err := sarama.NewConsumerGroup(c.cfg.Brokers, c.cfg.Group, config)
	if err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	c.consumer = consumerGroup

	// 注册处理器
	c.mu.Lock()
	c.handlers[topic] = handler
	c.mu.Unlock()

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// 启动消费者
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 消费消息
			if err := consumerGroup.Consume(ctx, []string{topic}, &consumerGroupHandler{consumer: c}); err != nil {
				logx.Errorf("Error from consumer: %v", err)
			}
		}
	}()

	return nil
}

// Close 关闭消费者
func (c *KafkaConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	c.wg.Wait()

	if c.consumer != nil {
		return c.consumer.Close()
	}

	return nil
}

// consumerGroupHandler 实现了sarama.ConsumerGroupHandler接口
type consumerGroupHandler struct {
	consumer *KafkaConsumer
}

// Setup 在消费者会话开始时调用
func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 在消费者会话结束时调用
func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 处理分配给消费者的消息
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		topic := message.Topic

		// 获取处理器
		h.consumer.mu.Lock()
		handler, ok := h.consumer.handlers[topic]
		h.consumer.mu.Unlock()

		if !ok {
			logx.Errorf("No handler registered for topic: %s", topic)
			continue
		}

		// 处理消息
		key := string(message.Key)
		if err := handler(key, message.Value); err != nil {
			logx.Errorf("Failed to handle message: %v", err)
		}

		// 标记消息为已处理
		session.MarkMessage(message, "")
	}

	return nil
}
