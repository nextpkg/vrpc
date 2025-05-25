package kq

import (
	"context"
	"errors"
	"sync"

	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-queue/kq"
)

// KqPlugin 是基于go-queue/kq的Kafka插件实现
type KqPlugin struct {
	config *rpcbus.KqConfig
	mu     sync.RWMutex
}

// NewKqPlugin 创建一个新的Kq插件
func NewKqPlugin() *KqPlugin {
	return &KqPlugin{}
}

// Name 返回插件名称
func (p *KqPlugin) Name() string {
	return "kq"
}

// Register 注册插件
func (p *KqPlugin) Register(manager *rpcbus.PluginManager) {
	manager.RegisterPlugin(p)
}

// Init 初始化插件
func (p *KqPlugin) Init(cfg interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	kqConfig, ok := cfg.(*rpcbus.KqConfig)
	if !ok {
		return rpcbus.ErrInvalidConfig
	}

	if len(kqConfig.Brokers) == 0 {
		return errors.New("kafka brokers cannot be empty")
	}

	p.config = kqConfig
	return nil
}

// GetConfig 获取插件配置
func (p *KqPlugin) GetConfig() interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.config
}

// CreateProducer 创建Kafka消息生产者
func (p *KqPlugin) CreateProducer() (rpcbus.MessageProducer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil, errors.New("kq plugin not initialized")
	}

	pusher := kq.NewPusher(p.config.Brokers, p.config.Topic)
	return &KqProducer{pusher: pusher}, nil
}

// CreateConsumer 创建Kafka消息消费者
func (p *KqPlugin) CreateConsumer() (rpcbus.MessageConsumer, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.config == nil {
		return nil, errors.New("kq plugin not initialized")
	}

	return &KqConsumer{
		brokers: p.config.Brokers,
		group:   p.config.Group,
		topic:   p.config.Topic,
	}, nil
}

// KqProducer 基于go-queue/kq的消息生产者实现
type KqProducer struct {
	pusher *kq.Pusher
}

// Produce 发送消息到Kafka
func (p *KqProducer) Produce(ctx context.Context, key string, value []byte) error {
	return p.pusher.Push(string(value))
}

// Close 关闭生产者
func (p *KqProducer) Close() error {
	// go-queue/kq的Pusher没有提供Close方法，这里返回nil
	return nil
}
