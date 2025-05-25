package vrpc

import (
	"errors"
	"time"
)

// Config 是vRPC的主配置结构
type Config struct {
	// 插件名称，如"kafka"或"kq"
	PluginName string `json:"plugin_name" yaml:"plugin_name"`
	// 插件配置，根据PluginName确定具体类型
	PluginConfig interface{} `json:"plugin_config" yaml:"plugin_config"`
	// 请求主题
	RequestTopic string `json:"request_topic" yaml:"request_topic"`
	// 响应主题
	ResponseTopic string `json:"response_topic" yaml:"response_topic"`
	// 超时设置
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// Validate 验证配置是否有效
func (c *Config) Validate() error {
	if c.PluginName == "" {
		return errors.New("plugin name cannot be empty")
	}

	if c.RequestTopic == "" {
		return errors.New("request topic cannot be empty")
	}

	if c.ResponseTopic == "" {
		return errors.New("response topic cannot be empty")
	}

	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second // 设置默认超时时间
	}

	return nil
}
