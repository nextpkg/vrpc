package kq

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
)

// KqConsumer 基于go-queue/kq的消息消费者实现
type KqConsumer struct {
	brokers  []string
	group    string
	topic    string
	consumer *kq.Consumer
	cancel   context.CancelFunc
}

// Consume 从Kafka消费消息
func (c *KqConsumer) Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error {
	// 如果提供了topic参数，则使用该参数，否则使用配置中的topic
	if topic == "" {
		topic = c.topic
	}

	// 创建kq配置
	conf := kq.KqConf{
		ServiceConf: service.ServiceConf{
			Name: fmt.Sprintf("kq-consumer-%s", topic),
		},
		Brokers:   c.brokers,
		Group:     c.group,
		Topic:     topic,
		Offset:    "last", // 默认从最新位置开始消费
		Consumers: 8,      // 默认消费者数量
	}

	// 创建上下文，以便后续关闭
	consumerCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	// 创建消费者
	consumer := kq.MustNewQueue(conf, kq.WithHandle(func(k, v string) error {
		return handler(k, []byte(v))
	}))

	c.consumer = consumer

	// 启动消费者
	go func() {
		consumer.Start()
		<-consumerCtx.Done()
		consumer.Stop()
		logx.Info("KqConsumer stopped")
	}()

	return nil
}

// Close 关闭消费者
func (c *KqConsumer) Close() error {
	if c.cancel != nil {
		c.cancel()
	}

	if c.consumer != nil {
		c.consumer.Stop()
	}

	return nil
}
