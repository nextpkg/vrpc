package kafka

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-zero/core/logx"
)

// KafkaProducer 实现了MessageProducer接口，用于向Kafka发送消息
type KafkaProducer struct {
	cfg      *rpcbus.KafkaConfig
	producer sarama.SyncProducer
}

// NewKafkaProducer 创建一个新的Kafka生产者
func NewKafkaProducer(cfg *rpcbus.KafkaConfig) *KafkaProducer {
	return &KafkaProducer{
		cfg: cfg,
	}
}

// Init 初始化Kafka生产者
func (p *KafkaProducer) Init() error {
	// 创建Kafka配置
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true
	config.Version = sarama.V2_0_0_0

	// 创建生产者
	producer, err := sarama.NewSyncProducer(p.cfg.Brokers, config)
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}

	p.producer = producer
	return nil
}

// Produce 向指定的主题发送消息
func (p *KafkaProducer) Produce(ctx context.Context, key string, value []byte) error {
	// 确保生产者已初始化
	if p.producer == nil {
		if err := p.Init(); err != nil {
			return err
		}
	}

	// 创建消息
	msg := &sarama.ProducerMessage{
		Topic: p.cfg.Topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	// 发送消息
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	logx.Infof("Message sent to partition %d at offset %d", partition, offset)
	return nil
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}

	return nil
}
