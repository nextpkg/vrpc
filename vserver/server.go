package vserver

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/internal/proto/gen"
	"github.com/nextpkg/vrpc/rpcbus"
)

type Server struct {
	gen.UnimplementedVRPCServiceServer

	config          *vrpc.Config
	publisher       rpcbus.Publisher
	consumer        rpcbus.Consumer
	downstreamProxy *DownstreamProxy
	plugins         []Plugin
}

func NewServer(c *vrpc.Config) (*Server, error) {
	// 创建MQ发布者和消费者
	publisher, err := rpcbus.NewPublisher(&c.RpcBus, "vrpc-responses")
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	consumer, err := rpcbus.NewConsumer(&c.RpcBus, "vrpc-server", c.Topics.Request)
	if err != nil {
		publisher.Close()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	server := &Server{
		config:    c,
		publisher: publisher,
		consumer:  consumer,
	}

	// 初始化下游代理
	server.downstreamProxy = NewDownstreamProxy(c.Targets, publisher)

	// 初始化插件
	server.initPlugins()

	return server, nil
}

func (s *Server) Start() error {
	// 启动消息队列消费
	ctx := context.Background()
	if err := s.consumer.Consume(ctx, s.config.Topics.Request, s.handleRequest); err != nil {
		return fmt.Errorf("failed to start request consumer: %w", err)
	}

	logx.Info("vRPC server started successfully")

	// 启动gRPC服务
	server := zrpc.MustNewServer(s.config.RpcServerConf, func(grpcServer *grpc.Server) {
		gen.RegisterVRPCServiceServer(grpcServer, s)
	})

	server.Start()
	return nil
}

func (s *Server) Stop() {
	if s.publisher != nil {
		s.publisher.Close()
	}
	if s.consumer != nil {
		s.consumer.Close()
	}
}

// Call 处理gRPC调用(主要用于健康检查等)
func (s *Server) Call(ctx context.Context, req *gen.VRPCRequest) (*gen.VRPCResponse, error) {
	return s.processRequest(ctx, req)
}

// HealthCheck 健康检查
func (s *Server) HealthCheck(ctx context.Context, req *gen.HealthCheckRequest) (*gen.HealthCheckResponse, error) {
	return &gen.HealthCheckResponse{Status: gen.HealthCheckResponse_SERVING}, nil
}

// handleRequest 处理来自MQ的请求
func (s *Server) handleRequest(ctx context.Context, req *gen.VRPCRequest) error {
	response, err := s.processRequest(ctx, req)
	if err != nil {
		logx.Errorf("Failed to process request %s: %v", req.RequestId, err)
		// 发送错误响应
		errorResp := &gen.VRPCResponse{
			RequestId:    req.RequestId,
			StatusCode:   int32(codes.Internal),
			ErrorMessage: err.Error(),
			Timestamp:    time.Now().UnixNano(),
		}
		return s.publisher.PublishResponse(ctx, s.config.Topics.Response, errorResp)
	}

	// 发送成功响应
	return s.publisher.PublishResponse(ctx, s.config.Topics.Response, response)
}

// processRequest 处理请求并调用目标服务
func (s *Server) processRequest(ctx context.Context, req *gen.VRPCRequest) (*gen.VRPCResponse, error) {
	// 应用插件处理
	for _, plugin := range s.plugins {
		if err := plugin.BeforeRequest(ctx, req); err != nil {
			return nil, fmt.Errorf("plugin %s failed: %w", plugin.Name(), err)
		}
	}

	// 查找目标服务配置
	target, exists := s.config.Targets[req.ServiceName]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", req.ServiceName)
	}

	// 调用下游代理
	response, err := s.downstreamProxy.CallService(ctx, req, target)
	if err != nil {
		return nil, err
	}

	// 应用插件后处理
	for _, plugin := range s.plugins {
		if err := plugin.AfterResponse(ctx, req, response); err != nil {
			logx.Errorf("Plugin %s after response failed: %v", plugin.Name(), err)
		}
	}

	return response, nil
}

// initPlugins 初始化插件
func (s *Server) initPlugins() {
	// TODO: 根据配置加载插件
	// 这里可以实现插件加载逻辑
}
