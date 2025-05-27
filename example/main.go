package main

import (
	"context"
	"log"
	"time"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/vclient"
	"github.com/nextpkg/vrpc/vserver"
	"github.com/zeromicro/go-zero/zrpc"
)

func main() {
	// 示例1: 启动vRPC服务器
	go startServer()

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	// 示例2: 使用vRPC客户端
	useClient()
}

func startServer() {
	// 服务器配置
	config := &vrpc.Config{
		Name: "vrpc-server",
		Host: "0.0.0.0",
		Port: 8080,
		RpcServerConf: zrpc.RpcServerConf{
			ListenOn: "0.0.0.0:9090",
		},
		RpcBus: rpcbus.Config{
			Type:    "kafka",
			Brokers: []string{"localhost:9092"},
		},
		Topics: struct {
			Request  string `json:",default=vrpc.requests"`
			Response string `json:",default=vrpc.responses"`
		}{
			Request:  "vrpc.requests",
			Response: "vrpc.responses",
		},
		Targets: map[string]vrpc.Target{
			"demo": {
				Endpoint: "localhost:8081",
				Timeout:  "30s",
			},
		},
	}

	// 创建并启动服务器
	server, err := vserver.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Println("Starting vRPC server...")
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func useClient() {
	// 客户端配置
	config := &vclient.Config{
		ClientID: "demo-client",
		RpcBus: rpcbus.Config{
			Type:    "kafka",
			Brokers: []string{"localhost:9092"},
		},
		Topics: struct {
			Request  string `json:"request" yaml:"request"`
			Response string `json:"response" yaml:"response"`
		}{
			Request:  "vrpc.requests",
			Response: "vrpc.responses",
		},
		Debug: true,
	}
	config.Timeout.Request = 30 * time.Second
	config.Retry.Count = 3
	config.Retry.Interval = 100 * time.Millisecond

	// 初始化vRPC客户端
	if err := vrpc.InitClient(config); err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}
	defer vrpc.Close()

	// 方式1: 直接使用vRPC客户端
	client := vrpc.GetClient()
	ctx := context.Background()

	// 模拟请求数据
	requestData := []byte(`{"message": "Hello vRPC"}`)
	metadata := map[string]string{
		"user-id":  "12345",
		"trace-id": "abc-def-ghi",
	}

	response, err := client.Call(ctx, "demo", "Ping", requestData, metadata)
	if err != nil {
		log.Printf("Direct call failed: %v", err)
	} else {
		log.Printf("Direct call response: %s", string(response.Payload))
	}

	// 方式2: 使用gRPC拦截器（推荐方式）
	useGRPCInterceptor()
}

func useGRPCInterceptor() {
	// 初始化切面
	v := zrpc.WithUnaryClientInterceptor(vrpc.Interceptor())

	// 初始化gRPC客户端
	c := zrpc.MustNewClient(zrpc.RpcClientConf{
		Endpoints: []string{"localhost:9090"}, // 这里实际不会使用，因为被vRPC拦截
	}, v)

	// 这里应该是实际的gRPC客户端调用
	// 例如: resp, err := democlient.NewDemo(c).Ping(ctx, &democlient.Request{})
	log.Println("gRPC client with vRPC interceptor initialized successfully")
	log.Println("Client:", c)
}
