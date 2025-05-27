// pkg/mq/factory.go
package rpcbus

import (
	"fmt"
)

// NewPublisher 创建消息发布者
func NewPublisher(config *Config, topic string) (Publisher, error) {
	switch config.Type {
	case "kafka":
		return NewKafkaPublisher(config)
	default:
		return nil, fmt.Errorf("unsupported MQ type: %s", config.Type)
	}
}

// NewConsumer 创建消息消费者
func NewConsumer(config *Config, consumerGroup string, topic string) (Consumer, error) {
	switch config.Type {
	case "kafka":
		return NewKafkaConsumer(config, consumerGroup, topic)
	default:
		return nil, fmt.Errorf("unsupported MQ type: %s", config.Type)
	}
}
