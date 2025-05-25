package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nextpkg/vrpc/example/nextcfg"
	"github.com/nextpkg/vrpc/example/nextcfgclient"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/rpcbus/kafka"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

func main() {
	// 1. 加载配置
	cfg, err := rpcbus.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 2. 创建插件管理器
	pm := rpcbus.NewPluginManager()

	// 3. 注册 Kafka 插件，传入配置提供者函数
	kafka.Register(pm, func() *rpcbus.KafkaConfig {
		return cfg.Kafka
	})

	// 4. 初始化所有插件
	if err := pm.Initialize(); err != nil {
		log.Fatalf("初始化插件失败: %v", err)
	}
	kq.KqConf{}

	// 5. 获取 Kafka 插件
	kafkaPluginInterface, ok := pm.GetPlugin("kafka")
	if !ok {
		log.Fatalf("Kafka 插件未注册")
	}

	// 类型断言为 Kafka 插件类型
	kafkaPlugin, ok := kafkaPluginInterface.(*kafka.KafkaPlugin)
	if !ok {
		log.Fatalf("插件类型错误")
	}

	// 6. 获取 Kafka 配置
	kafkaConfig := kafkaPlugin.CurrentConfig()

	// 7. 使用配置
	fmt.Printf("Kafka 配置: Test = %d\n", kafkaConfig.Test)

	// 或者通过插件管理器的通用方法获取配置
	configInterface, err := pm.GetPluginConfig("kafka")
	if err != nil {
		log.Fatalf("获取 Kafka 配置失败: %v", err)
	}
	kafkaConfig2 := configInterface.(rpcbus.KafkaConfig)
	fmt.Printf("Kafka 配置 (通过管理器): Test = %d\n", kafkaConfig2.Test)
	return

	cc := zrpc.MustNewClient(zrpc.RpcClientConf{
		Target: "127.0.0.1:8080",
	})

	resp, err := nextcfgclient.NewNextcfg(cc).Ping(context.Background(), &nextcfg.Request{
		Ping: "ping",
	})
	if err != nil {
		panic(err)
	}

	logx.Alert(resp.Pong)
}
