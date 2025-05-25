package kafka

import (
	"context"
	"errors"

	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

// KafkaPlugin 实现了Kafka消息队列的插件
type KafkaPlugin struct {
	configProvider func() *rpcbus.KafkaConfig
	currentConfig  *rpcbus.KafkaConfig
}

// Register 注册Kafka插件到插件管理器
func Register(pm *rpcbus.PluginManager, configProvider func() *rpcbus.KafkaConfig) {
	pm.Register("kafka", func() rpcbus.Plugin {
		return &KafkaPlugin{
			configProvider: configProvider,
		}
	})
}

// Init 初始化Kafka插件
func (p *KafkaPlugin) Init() error {
	p.currentConfig = p.configProvider()
	if p.currentConfig == nil {
		return errors.New("kafka config is nil")
	}
	return nil
}

// GetConfig 获取当前Kafka配置
func (p *KafkaPlugin) GetConfig() interface{} {
	return *p.currentConfig
}

// CurrentConfig 获取当前Kafka配置
func (p *KafkaPlugin) CurrentConfig() *rpcbus.KafkaConfig {
	return p.currentConfig
}

// KafkaProducer 实现了消息生产者接口
type KafkaProducer struct {
	pusher *kq.Pusher
}

// NewKafkaProducer 创建一个新的Kafka生产者
func NewKafkaProducer(cfg *rpcbus.KafkaConfig) *KafkaProducer {
	pusher := kq.NewPusher(cfg.Brokers, cfg.Topic)
	return &KafkaProducer{
		pusher: pusher,
	}
}

// Produce 发送消息到Kafka
func (p *KafkaProducer) Produce(ctx context.Context, key string, value []byte) error {
	return p.pusher.Push(string(value))
}

// Close 关闭Kafka生产者
func (p *KafkaProducer) Close() error {
	return nil
}

// KafkaConsumer 实现了消息消费者接口
type KafkaConsumer struct {
	consumer *kq.Consumer
}

// NewKafkaConsumer 创建一个新的Kafka消费者
func NewKafkaConsumer(cfg *rpcbus.KafkaConfig) *KafkaConsumer {
	c := kq.MustNewConsumer(kq.KqConf{
		Brokers:   cfg.Brokers,
		Group:     cfg.Group,
		Topic:     cfg.Topic,
		Offset:    "newest",
		ConsumeBy: kq.ConsumeByPartition,
	})

	return &KafkaConsumer{
		consumer: c,
	}
}

// Consume 从Kafka消费消息
func (c *KafkaConsumer) Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error {
	err := c.consumer.Start(func(_, v string) error {
		return handler("", []byte(v))
	})

	if err != nil {
		logx.Errorf("Failed to start consumer: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}
