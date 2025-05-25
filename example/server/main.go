package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/rpcbus/kafka"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// 示例配置结构
type Config struct {
	RPC    zrpc.RpcServerConf
	VRPC   vrpc.Config
	Kafka  rpcbus.KafkaConfig
	Server struct {
		Name string
	}
}

// 示例服务实现
type ExampleService struct {
	// 这里可以添加你的服务实现所需的字段
}

// 示例方法实现
func (s *ExampleService) Ping(ctx context.Context, req *PingRequest) (*PingResponse, error) {
	return &PingResponse{
		Message: fmt.Sprintf("Pong: %s", req.Message),
	}, nil
}

// 示例请求结构
type PingRequest struct {
	Message string
}

// 示例响应结构
type PingResponse struct {
	Message string
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

	// 创建vRPC配置
	c.VRPC.PluginName = "kafka"
	c.VRPC.PluginConfig = &c.Kafka
	c.VRPC.RequestTopic = "request-topic"
	c.VRPC.ResponseTopic = "response-topic"

	// 获取插件
	plugin, err := pluginManager.GetPlugin("kafka")
	if err != nil {
		log.Fatalf("Failed to get Kafka plugin: %v", err)
	}

	// 创建消费者
	consumer, err := plugin.CreateConsumer()
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer: %v", err)
	}

	// 创建生产者
	producer, err := plugin.CreateProducer()
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}

	// 创建编解码器
	coder := vrpc.NewCoder()

	// 创建下游代理
	downstreamProxy := vrpc.NewDownstreamProxy(consumer, producer, coder, c.VRPC.ResponseTopic)

	// 创建服务实例
	service := &ExampleService{}

	// 注册服务处理器
	downstreamProxy.RegisterHandler("ExampleService", "Ping", func(ctx context.Context, reqData interface{}) (interface{}, error) {
		// 转换请求数据
		req, ok := reqData.(*PingRequest)
		if !ok {
			return nil, fmt.Errorf("invalid request type: %T", reqData)
		}

		// 调用服务方法
		return service.Ping(ctx, req)
	})

	// 创建zRPC服务器
	srv := zrpc.MustNewServer(c.RPC, func(grpcServer *grpc.Server) {
		// 这里可以注册你的gRPC服务
		// 例如：pb.RegisterYourServiceServer(grpcServer, service)
	})

	// 添加服务组
	serviceGroup := service.NewServiceGroup()
	serviceGroup.Add(srv)

	// 启动下游代理
	ctx := context.Background()
	go func() {
		err := downstreamProxy.Start(ctx)
		if err != nil {
			log.Fatalf("Failed to start downstream proxy: %v", err)
		}
	}()

	fmt.Println("vRPC server starting...")

	// 启动服务
	serviceGroup.Start()
}
