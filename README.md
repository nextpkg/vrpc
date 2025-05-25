# vRPC

vRPC是一个通过消息队列将同步RPC调用转换为异步调用的中间件，支持多种RPC框架和消息队列。

## 特性

- 支持多种RPC框架（如go-zero）
- 支持多种消息队列（如Kafka）
- 任务级别的全局单例
- 故障转移
- 可扩展的插件系统

## 安装

```bash
go get github.com/nextpkg/vrpc
```

## 快速开始

### 配置

创建配置文件 `config.yaml`：

```yaml
Name: vrpc-example
Environment: dev

RPC:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: vrpc.rpc
  ListenOn: 127.0.0.1:8080
  Timeout: 5000

VRPC:
  Request:
    Topic: request-topic
    Kafka:
      Brokers:
        - localhost:9092
      Group: vrpc-request-group
  Respond:
    Topic: response-topic
    Kafka:
      Brokers:
        - localhost:9092
      Group: vrpc-response-group
  Timeout: 5000

Kafka:
  Brokers:
    - localhost:9092
  Group: vrpc-group
  Topic: vrpc-topic
```

### 客户端

```go
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/rpcbus/kafka"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
)

// 配置结构
type Config struct {
	RPC    zrpc.RpcClientConf
	VRPC   vrpc.Config
	Kafka  rpcbus.KafkaConfig
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

	// 创建vRPC客户端
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

	// 创建服务客户端
	// client := pb.NewYourServiceClient(zrpcClient.Conn())

	// 执行RPC调用
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// resp, err := client.YourMethod(ctx, &pb.YourRequest{})
}
```

### 服务端

```go
package main

import (
	"context"
	"flag"
	"log"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/rpcbus/kafka"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// 配置结构
type Config struct {
	RPC    zrpc.RpcServerConf
	VRPC   vrpc.Config
	Kafka  rpcbus.KafkaConfig
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

	// 创建Kafka消费者和生产者
	consumer := kafka.NewKafkaConsumer(&rpcbus.KafkaConfig{
		Brokers: c.Kafka.Brokers,
		Group:   c.Kafka.Group,
		Topic:   c.VRPC.Request.Topic,
	})

	producer := kafka.NewKafkaProducer(&rpcbus.KafkaConfig{
		Brokers: c.Kafka.Brokers,
		Group:   c.Kafka.Group,
		Topic:   c.VRPC.Respond.Topic,
	})
	err = producer.Init()
	if err != nil {
		log.Fatalf("Failed to init Kafka producer: %v", err)
	}

	// 创建下游代理
	downstreamProxy := vrpc.NewDownstreamProxy(consumer, producer, c.VRPC.Respond.Topic)

	// 注册服务处理器
	// downstreamProxy.RegisterHandler("YourService.YourMethod", handler)

	// 创建zRPC服务器
	srv := zrpc.MustNewServer(c.RPC, func(grpcServer *grpc.Server) {
		// 注册gRPC服务
		// pb.RegisterYourServiceServer(grpcServer, service)
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

	// 启动服务
	serviceGroup.Start()
}
```

## 架构

vRPC通过以下组件实现RPC调用的异步化：

1. **客户端拦截器**：拦截RPC调用并将其重定向到vRPC客户端
2. **上游代理**：将RPC请求转换为消息并发送到消息队列
3. **下游代理**：从消息队列接收消息并转换为RPC调用
4. **响应连接**：将RPC响应发送回客户端

## 插件系统

vRPC支持通过插件系统扩展消息队列的支持：

1. **插件管理器**：管理和注册插件
2. **插件接口**：定义插件的行为
3. **消息生产者**：发送消息到消息队列
4. **消息消费者**：从消息队列接收消息

## 贡献

欢迎贡献代码和提出问题！请遵循以下步骤：

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建Pull Request

## 许可证

本项目采用MIT许可证 - 详见 [LICENSE](LICENSE) 文件