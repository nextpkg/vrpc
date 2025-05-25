package vrpc

import (
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

// ClientInterceptor 截获客户端的RPC调用
type ClientInterceptor struct {
	vrpcClient *Client
}

func NewClientInterceptor(cfg *Config) *ClientInterceptor {
	return &ClientInterceptor{
		vrpcClient: NewClient(NewUpstreamProxy(cfg)),
	}
}

// UnaryInterceptor 拦截RPC调用并将其重定向到vRPC
func (ci *ClientInterceptor) UnaryInterceptor(ctx context.Context, method string, req, reply any) error {
	// 将请求转发到vRPC客户端
	return ci.vrpcClient.Call(ctx, method, req, reply)
}

func NewZRPCUnaryInterceptor() zrpc.ClientOption {
	c := ClientInterceptor{}
	return zrpc.WithUnaryClientInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return c.UnaryInterceptor(ctx, method, req, reply)
	})
}
