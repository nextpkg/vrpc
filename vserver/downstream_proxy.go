package vserver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nextpkg/vrpc"
	"github.com/nextpkg/vrpc/internal/proto/gen"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DownstreamProxy 下游代理，负责调用实际的RPC服务
type DownstreamProxy struct {
	connections map[string]*grpc.ClientConn
	mu          sync.RWMutex
	publisher   rpcbus.Publisher
}

// NewDownstreamProxy 创建下游代理
func NewDownstreamProxy(targets map[string]vrpc.Target, publisher rpcbus.Publisher) *DownstreamProxy {
	proxy := &DownstreamProxy{
		connections: make(map[string]*grpc.ClientConn),
		publisher:   publisher,
	}

	// 建立到各个服务的连接
	for serviceName, target := range targets {
		if conn, err := grpc.Dial(target.Endpoint, grpc.WithInsecure()); err == nil {
			proxy.connections[serviceName] = conn
			logx.Infof("Connected to service %s at %s", serviceName, target.Endpoint)
		} else {
			logx.Errorf("Failed to connect to service %s at %s: %v", serviceName, target.Endpoint, err)
		}
	}

	return proxy
}

// CallService 调用目标服务
func (dp *DownstreamProxy) CallService(ctx context.Context, req *gen.VRPCRequest, target vrpc.Target) (*gen.VRPCResponse, error) {
	dp.mu.RLock()
	conn, exists := dp.connections[req.ServiceName]
	dp.mu.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "service not found: %s", req.ServiceName)
	}

	// 设置超时
	timeout, err := time.ParseDuration(target.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 构造gRPC方法名
	fullMethod := fmt.Sprintf("/%s/%s", req.ServiceName, req.MethodName)

	// 调用gRPC服务
	var responsePayload []byte
	err = conn.Invoke(ctx, fullMethod, req.Payload, &responsePayload)

	// 构造响应
	response := &gen.VRPCResponse{
		RequestId: req.RequestId,
		Timestamp: time.Now().UnixNano(),
		Metadata:  make(map[string]string),
	}

	if err != nil {
		if grpcStatus, ok := status.FromError(err); ok {
			response.StatusCode = int32(grpcStatus.Code())
			response.ErrorMessage = grpcStatus.Message()
		} else {
			response.StatusCode = int32(codes.Internal)
			response.ErrorMessage = err.Error()
		}
	} else {
		response.StatusCode = int32(codes.OK)
		response.Payload = responsePayload
	}

	return response, nil
}

// Close 关闭所有连接
func (dp *DownstreamProxy) Close() error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	for serviceName, conn := range dp.connections {
		if err := conn.Close(); err != nil {
			logx.Errorf("Failed to close connection to %s: %v", serviceName, err)
		}
	}
	dp.connections = make(map[string]*grpc.ClientConn)
	return nil
}
