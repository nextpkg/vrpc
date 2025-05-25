// package rpcbus 提供了一个插件系统，用于动态加载不同插件的配置
package rpcbus

import (
	"errors"
	"sync"

	"github.com/zeromicro/go-zero/core/conf"
)

// Plugin 接口定义了插件的行为
type Plugin interface {
	// Init 初始化插件（可能会使用配置）
	Init() error
	// GetConfig 获取最新的配置
	GetConfig() interface{}
}

// PluginFactory 定义了创建插件的工厂函数类型
type PluginFactory func() Plugin

// PluginManager 管理所有注册的插件
type PluginManager struct {
	factories map[string]PluginFactory
	plugins   map[string]Plugin
	mu        sync.RWMutex
}

// NewPluginManager 创建一个新的插件管理器
func NewPluginManager() *PluginManager {
	return &PluginManager{
		factories: make(map[string]PluginFactory),
		plugins:   make(map[string]Plugin),
	}
}

// Register 注册一个插件工厂
func (pm *PluginManager) Register(name string, factory PluginFactory) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.factories[name] = factory
}

// GetPlugin 获取指定名称的插件
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	plugin, ok := pm.plugins[name]
	return plugin, ok
}

// Initialize 初始化所有已注册的插件
func (pm *PluginManager) Initialize() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, factory := range pm.factories {
		plugin := factory()
		err := plugin.Init()
		if err != nil {
			return err
		}
		pm.plugins[name] = plugin
	}

	return nil
}

// GetPluginConfig 获取指定插件的最新配置
func (pm *PluginManager) GetPluginConfig(name string) (interface{}, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, ok := pm.plugins[name]
	if !ok {
		return nil, errors.New("plugin not found: " + name)
	}

	return plugin.GetConfig(), nil
}

// Config 是主配置结构
type Config struct {
	// 空结构作为基础
}

// KafkaConfig 是 Kafka 插件的配置
type KafkaConfig struct {
	Test int `json:"test" yaml:"test"`
}

// PluginConfig 包含所有插件的配置
type PluginConfig struct {
	Config
	Kafka *KafkaConfig `json:"kafka" yaml:"kafka"`
}

// LoadConfig 加载配置文件
func LoadConfig(file string) (*PluginConfig, error) {
	var cfg PluginConfig
	err := conf.Load(file, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
