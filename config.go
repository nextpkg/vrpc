package vrpc

import (
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	// 服务配置
	Name string `json:",default=vrpc-server"`
	Host string `json:",default=0.0.0.0"`
	Port int    `json:",default=8080"`

	// gRPC服务配置
	RpcServerConf zrpc.RpcServerConf

	// REST API配置
	RestConf rest.RestConf

	// MQ配置
	RpcBus rpcbus.Config

	// Topic配置
	Topics struct {
		Request  string `json:",default=vrpc.requests"`
		Response string `json:",default=vrpc.responses"`
	}

	// 目标RPC服务配置
	Targets map[string]Target

	// 插件配置
	Plugins []PluginConfig
}

type Target struct {
	Endpoint string            `json:"endpoint"`
	Timeout  string            `json:"timeout,default=30s"`
	Settings map[string]string `json:"settings"`
}

type PluginConfig struct {
	Name     string                 `json:"name"`
	Enabled  bool                   `json:"enabled,default=true"`
	Settings map[string]interface{} `json:"settings"`
}
