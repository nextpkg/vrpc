package kafka

import (
	"context"
	"errors"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

// KafkaPlugin Kafka插件实现
type KafkaPlugin struct {
	config *rpcbus.KafkaConfig
	mu     sync.RWMutex
}

// NewKafkaPlugin 创建一个新的Kafka插件
func NewKafkaPlugin() *KafkaPlugin {
	return &KafkaPlugin{}
}

// Name 返回插件名称
func (p *KafkaPlugin) Name() string {
	return "kafka"
}

// Register 注册插件
func (p *KafkaPlugin) Register(manager *rpcbus.PluginManager) {
	manager.RegisterPlugin(p)
}

// Init 初始化插件
func (p *KafkaPlugin) Init(cfg interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	kafkaConfig, ok := cfg.(*rpcbus.KafkaConfig)
	if !ok {
		return rpcbus.ErrInvalidConfig
	}

	if len(kafkaConfig.Brokers) == 0 {
		return errors.New("kafka brokers cannot be empty")
	}

	p.config = kafkaConfig
	return nil
}

// GetConfig 获取插件配置
func (p *KafkaPlugin) GetConfig() interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.config
}

// CreateProducer 创建Kafka消息生产者
func (p *KafkaPlugin) CreateProducer() (rpcbus.MessageProducer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil, errors.New("kafka plugin not initialized")
	}

	pusher := kq.NewPusher(p.config.Brokers, p.config.Topic)
	return &KafkaProducer{pusher: pusher}, nil
}

// CreateConsumer 创建Kafka消息消费者
func (p *KafkaPlugin) CreateConsumer() (rpcbus.MessageConsumer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil, errors.New("kafka plugin not initialized")
	}

	return &KafkaConsumer{
		brokers: p.config.Brokers,
		group:   p.config.Group,
	}, nil
}

// KafkaProducer Kafka消息生产者实现
type KafkaProducer struct {
	pusher *kq.Pusher
}

// Produce 发送消息到Kafka
func (p *KafkaProducer) Produce(ctx context.Context, key string, value []byte) error {
	return p.pusher.Push(string(value))
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	return nil
}

// KafkaConsumer Kafka消息消费者实现
type KafkaConsumer struct {
	brokers  []string
	group    string
	consumer sarama.ConsumerGroup
	cancel   context.CancelFunc
}

// Consume 从Kafka消费消息
func (c *KafkaConsumer) Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	var err error
	c.consumer, err = sarama.NewConsumerGroup(c.brokers, c.group, config)
	if err != nil {
		return err
	}

	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go func() {
		for {
			if err := c.consumer.Consume(consumerCtx, []string{topic}, &consumerHandler{handler: handler}); err != nil {
				logx.Errorf("Error from consumer: %v", err)
				if consumerCtx.Err() != nil {
					// 上下文已取消，退出循环
					break
				}
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

	if c.consumer != nil {
		return c.consumer.Close()
	}

	return nil
}

// consumerHandler 实现sarama.ConsumerGroupHandler接口
type consumerHandler struct {
	handler func(key string, value []byte) error
}

func (h *consumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		if err := h.handler(string(message.Key), message.Value); err != nil {
			logx.Errorf("Error handling message: %v", err)
		}
		session.MarkMessage(message, "")
	}

	return nil
}
