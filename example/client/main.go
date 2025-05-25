package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/rpcbus/kafka"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
)

// 示例配置结构
type Config struct {
	RPC    zrpc.RpcClientConf
	VRPC   vrpc.Config
	Kafka  rpcbus.KafkaConfig
	Server struct {
		Name string
	}
}

var configFile = flag.String("f", "config.yaml", "配置文件路径")

func main() {
	flag.Parse()

	// 加载配置
	var c Config
	conf.MustLoad(*configFile, &c)

	// 注册Kafka插件
	pluginManager := rpcbus.NewPluginManager()
	kafkaPlugin := kafka.NewKafkaPlugin()
	kafkaPlugin.Register(pluginManager)

	// 初始化Kafka插件
	err := pluginManager.InitPlugin("kafka", &c.Kafka)
	if err != nil {
		log.Fatalf("Failed to init Kafka plugin: %v", err)
	}

	// 获取Kafka配置
	kafkaConfig, ok := kafkaPlugin.GetConfig().(*rpcbus.KafkaConfig)
	if !ok {
		log.Fatal("Failed to get Kafka config")
	}

	// 创建vRPC客户端
	c.VRPC.Request.Kafka.Brokers = kafkaConfig.Brokers
	c.VRPC.Request.Kafka.Group = kafkaConfig.Group
	c.VRPC.Request.Topic = "request-topic"
	c.VRPC.Respond.Kafka.Brokers = kafkaConfig.Brokers
	c.VRPC.Respond.Kafka.Group = kafkaConfig.Group
	c.VRPC.Respond.Topic = "response-topic"
	c.VRPC.Timeout = time.Second * 5

	vrpcClient, err := vrpc.NewClient(&c.VRPC)
	if err != nil {
		log.Fatalf("Failed to create vRPC client: %v", err)
	}

	// 创建zRPC客户端
	zrpcClient := zrpc.MustNewClient(c.RPC)

	// 注册拦截器
	err = vrpc.RegisterInterceptor(vrpcClient, zrpcClient)
	if err != nil {
		log.Fatalf("Failed to register interceptor: %v", err)
	}

	// 创建gRPC客户端
	conn := zrpcClient.Conn()

	// 这里可以创建你的服务客户端
	// 例如：client := pb.NewYourServiceClient(conn)

	// 执行RPC调用
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// 这里可以执行你的RPC调用
	// 例如：resp, err := client.YourMethod(ctx, &pb.YourRequest{})

	fmt.Println("vRPC client example completed")
}
