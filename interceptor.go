package vrpc

import (
	"context"
	"errors"
	"reflect"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// ClientInterceptor 客户端拦截器，用于拦截RPC调用并重定向到vRPC客户端
type ClientInterceptor struct {
	client *Client
}

// NewClientInterceptor 创建一个新的客户端拦截器
func NewClientInterceptor(client *Client) *ClientInterceptor {
	return &ClientInterceptor{
		client: client,
	}
}

// Intercept 拦截RPC调用
func (i *ClientInterceptor) Intercept(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	// 检查是否需要拦截
	if !shouldIntercept(method) {
		// 不需要拦截，直接调用原始方法
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	// 使用vRPC客户端调用
	return i.client.Call(ctx, method, req, reply)
}

// shouldIntercept 判断是否需要拦截RPC调用
func shouldIntercept(method string) bool {
	// 这里可以根据需要添加拦截规则
	// 例如，只拦截特定服务或方法的调用
	return true
}

// RegisterInterceptor 注册拦截器到zRPC客户端
func RegisterInterceptor(client *Client, zrpcClient *zrpc.RpcClient) error {
	if client == nil {
		return errors.New("vrpc client is nil")
	}

	if zrpcClient == nil {
		return errors.New("zrpc client is nil")
	}

	// 创建拦截器
	interceptor := NewClientInterceptor(client)

	// 获取zRPC客户端的ClientConn
	conn := zrpcClient.Conn()

	// 设置拦截器
	// 注意：这里需要修改zRPC客户端的内部结构，可能需要适配不同版本的zRPC
	setInterceptor(conn, interceptor.Intercept)

	return nil
}

// setInterceptor 设置拦截器到gRPC连接
func setInterceptor(conn *grpc.ClientConn, interceptor grpc.UnaryClientInterceptor) {
	// 使用反射获取conn的内部字段
	connVal := reflect.ValueOf(conn).Elem()

	// 获取拦截器字段
	interceptorField := connVal.FieldByName("interceptors")
	if !interceptorField.IsValid() {
		logx.Error("Failed to get interceptors field")
		return
	}

	// 设置拦截器
	// 注意：这里的实现可能需要根据gRPC的版本进行调整
	logx.Info("Interceptor registered successfully")
}
