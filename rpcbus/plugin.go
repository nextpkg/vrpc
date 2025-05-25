package rpcbus

import (
	"context"
	"errors"
	"sync"
)

// 错误定义
var (
	ErrPluginNotFound = errors.New("plugin not found")
	ErrInvalidConfig  = errors.New("invalid plugin configuration")
)

// MessageHandler 消息处理函数类型
type MessageHandler func(key string, value []byte) error

// MessageProducer 消息生产者接口
type MessageProducer interface {
	// Produce 发送消息到指定主题
	Produce(ctx context.Context, key string, value []byte) error
	// Close 关闭生产者
	Close() error
}

// MessageConsumer 消息消费者接口
type MessageConsumer interface {
	// Consume 从指定主题消费消息
	Consume(ctx context.Context, topic string, handler func(key string, value []byte) error) error
	// Close 关闭消费者
	Close() error
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers []string `json:"brokers"`
	Group   string   `json:"group"`
	Topic   string   `json:"topic"`
}

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string
	// Register 注册插件
	Register(manager *PluginManager)
	// Init 初始化插件
	Init(cfg interface{}) error
	// GetConfig 获取插件配置
	GetConfig() interface{}
	// CreateProducer 创建消息生产者
	CreateProducer() (MessageProducer, error)
	// CreateConsumer 创建消息消费者
	CreateConsumer() (MessageConsumer, error)
}

// PluginManager 插件管理器
type PluginManager struct {
	plugins map[string]Plugin
	mu      sync.RWMutex
}

// NewPluginManager 创建一个新的插件管理器
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]Plugin),
	}
}

// RegisterPlugin 注册插件
func (m *PluginManager) RegisterPlugin(plugin Plugin) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.plugins[plugin.Name()] = plugin
}

// GetPlugin 获取插件
func (m *PluginManager) GetPlugin(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, ok := m.plugins[name]
	if !ok {
		return nil, ErrPluginNotFound
	}

	return plugin, nil
}

// InitPlugin 初始化插件
func (m *PluginManager) InitPlugin(name string, cfg interface{}) error {
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return err
	}

	return plugin.Init(cfg)
}
