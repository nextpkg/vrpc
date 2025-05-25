// package rpcbus 提供了一个插件系统，用于动态加载不同插件的配置
package rpcbus

import (
	"errors"

	"github.com/zeromicro/go-zero/core/conf"
)

// PluginFactory 定义了创建插件的工厂函数类型
type PluginFactory func() Plugin

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
	Brokers []string `json:"brokers" yaml:"brokers"`
	Group   string   `json:"group" yaml:"group"`
	Topic   string   `json:"topic" yaml:"topic"`
}

// KqConfig 是 go-queue/kq 插件的配置
type KqConfig struct {
	Brokers []string `json:"brokers" yaml:"brokers"`
	Group   string   `json:"group" yaml:"group"`
	Topic   string   `json:"topic" yaml:"topic"`
	Offset  string   `json:"offset,optional" yaml:"offset,optional"` // first或last，默认last
}

// PluginConfig 包含所有插件的配置
type PluginConfig struct {
	Config
	Kafka *KafkaConfig `json:"kafka" yaml:"kafka"`
	Kq    *KqConfig    `json:"kq" yaml:"kq"`
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
