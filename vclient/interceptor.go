package vclient

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nextpkg/vrpc/codec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	clientMu sync.RWMutex
)

// SetGlobalClient 设置全局客户端实例
func SetGlobalClient(client *Client) {
	clientMu.Lock()
	defer clientMu.Unlock()
	globalClient = client
}

// GetGlobalClient 获取全局客户端实例
func GetGlobalClient() *Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	return globalClient
}

// Interceptor 返回vRPC拦截器
func Interceptor() grpc.UnaryClientInterceptor {
	client := GetGlobalClient()
	if client == nil {
		panic("vRPC client not initialized, call vrpc.MustInitClient() first")
	}
	return client.UnaryInterceptor()
}

// UnaryInterceptor 创建gRPC一元拦截器
func (c *Client) UnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 解析方法名
		serviceName, methodName, err := parseGRPCMethod(method)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid method: %v", err)
		}

		// 序列化请求
		payload, err := codec.Serialize(req)
		if err != nil {
			return status.Errorf(codes.Internal, "failed to serialize request: %v", err)
		}

		// 提取元数据
		metadata := extractMetadata(ctx)

		// 通过vRPC发送请求
		response, err := c.Call(ctx, serviceName, methodName, payload, metadata)
		if err != nil {
			return err
		}

		// 检查响应状态
		if response.StatusCode != 0 {
			return status.Errorf(codes.Code(response.StatusCode), response.ErrorMessage)
		}

		// 反序列化响应
		if err := codec.Deserialize(response.Payload, reply); err != nil {
			return status.Errorf(codes.Internal, "failed to deserialize response: %v", err)
		}

		return nil
	}
}

// parseGRPCMethod 解析gRPC方法名
func parseGRPCMethod(fullMethod string) (string, string, error) {
	if !strings.HasPrefix(fullMethod, "/") {
		return "", "", fmt.Errorf("invalid method format: %s", fullMethod)
	}

	parts := strings.Split(strings.TrimPrefix(fullMethod, "/"), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid method format: %s", fullMethod)
	}

	// 提取服务名
	serviceParts := strings.Split(parts[0], ".")
	serviceName := serviceParts[len(serviceParts)-1]
	methodName := parts[1]

	return serviceName, methodName, nil
}

// extractMetadata 从上下文提取元数据
func extractMetadata(ctx context.Context) map[string]string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return nil
	}

	result := make(map[string]string)
	for key, values := range md {
		if len(values) > 0 {
			result[key] = values[0]
		}
	}

	return result
}
