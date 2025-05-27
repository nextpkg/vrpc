package vclient

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nextpkg/vrpc/rpcbus"
	"gopkg.in/yaml.v3"
)

// Config 客户端配置
type Config struct {
	// 客户端ID
	ClientID string `json:"client_id" yaml:"client_id"`

	// RpcBus配置
	RpcBus rpcbus.Config `json:"rpc_bus" yaml:"rpc_bus"`

	// 超时配置
	Timeout struct {
		Request time.Duration `json:"request" yaml:"request"`
		Connect time.Duration `json:"connect" yaml:"connect"`
	} `json:"timeout" yaml:"timeout"`

	// 重试配置
	Retry struct {
		Count    int           `json:"count" yaml:"count"`
		Interval time.Duration `json:"interval" yaml:"interval"`
	} `json:"retry" yaml:"retry"`

	// Topic配置
	Topics struct {
		Request  string `json:"request" yaml:"request"`
		Response string `json:"response" yaml:"response"`
	} `json:"topics" yaml:"topics"`

	// 调试开关
	Debug bool `json:"debug" yaml:"debug"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		ClientID: generateClientID(),
		RpcBus: rpcbus.Config{
			Type:    "kafka",
			Brokers: []string{"localhost:9092"},
			Settings: map[string]interface{}{
				"compression": "gzip",
				"batch_size":  100,
			},
		},
		Timeout: struct {
			Request time.Duration `json:"request" yaml:"request"`
			Connect time.Duration `json:"connect" yaml:"connect"`
		}{
			Request: 30 * time.Second,
			Connect: 5 * time.Second,
		},
		Retry: struct {
			Count    int           `json:"count" yaml:"count"`
			Interval time.Duration `json:"interval" yaml:"interval"`
		}{
			Count:    3,
			Interval: 100 * time.Millisecond,
		},
		Topics: struct {
			Request  string `json:"request" yaml:"request"`
			Response string `json:"response" yaml:"response"`
		}{
			Request:  "vrpc.requests",
			Response: "vrpc.responses",
		},
		Debug: false,
	}
}

// LoadFromFile 从文件加载配置
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv() *Config {
	config := DefaultConfig()

	if brokers := os.Getenv("VRPC_KAFKA_BROKERS"); brokers != "" {
		config.RpcBus.Brokers = strings.Split(brokers, ",")
	}

	if clientID := os.Getenv("VRPC_CLIENT_ID"); clientID != "" {
		config.ClientID = clientID
	}

	if timeout := os.Getenv("VRPC_REQUEST_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			config.Timeout.Request = duration
		}
	}

	return config
}

func generateClientID() string {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d-%d", hostname, pid, time.Now().Unix())
}
