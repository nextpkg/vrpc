package vclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nextpkg/vrpc/internal/proto/gen"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Client vRPC客户端
type Client struct {
	config       *Config
	publisher    rpcbus.Publisher
	consumer     rpcbus.Consumer
	pendingCalls sync.Map
	ctx          context.Context
	cancel       context.CancelFunc
	closeOnce    sync.Once
}

// pendingCall 等待中的调用
type pendingCall struct {
	requestID string
	respChan  chan *gen.VRPCResponse
	timer     *time.Timer
	ctx       context.Context
}

// NewClient 创建新的vRPC客户端
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建发布者
	publisher, err := rpcbus.NewPublisher(&config.RpcBus, config.Topics.Request)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	// 创建消费者
	consumer, err := rpcbus.NewConsumer(&config.RpcBus, fmt.Sprintf("vrpc-client-%s", config.ClientID), config.Topics.Response)
	if err != nil {
		publisher.Close()
		cancel()
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	client := &Client{
		config:    config,
		publisher: publisher,
		consumer:  consumer,
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动响应处理
	if err := client.startResponseHandler(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to start response handler: %w", err)
	}

	return client, nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		c.cancel()
		if c.publisher != nil {
			c.publisher.Close()
		}
		if c.consumer != nil {
			c.consumer.Close()
		}
	})
	return nil
}

// startResponseHandler 启动响应处理器
func (c *Client) startResponseHandler() error {
	return c.consumer.ConsumeResponse(c.ctx, c.config.Topics.Response, c.handleResponse)
}

// handleResponse 处理响应消息
func (c *Client) handleResponse(ctx context.Context, response *gen.VRPCResponse) error {
	if call, ok := c.pendingCalls.Load(response.RequestId); ok {
		pendingCall := call.(*pendingCall)
		select {
		case pendingCall.respChan <- response:
		default:
			// 通道已满或已关闭，忽略
		}
	}
	return nil
}

// Call 发起同步RPC调用
func (c *Client) Call(ctx context.Context, serviceName, methodName string, payload []byte, metadata map[string]string) (*gen.VRPCResponse, error) {
	requestID := uuid.New().String()

	// 创建vRPC请求
	req := &gen.VRPCRequest{
		RequestId:   requestID,
		ServiceName: serviceName,
		MethodName:  methodName,
		Payload:     payload,
		Metadata:    metadata,
		Timestamp:   time.Now().UnixNano(),
		TimeoutMs:   int32(c.config.Timeout.Request.Milliseconds()),
		ClientId:    c.config.ClientID,
	}

	// 创建等待响应的调用
	call := &pendingCall{
		requestID: requestID,
		respChan:  make(chan *gen.VRPCResponse, 1),
		timer:     time.NewTimer(c.config.Timeout.Request),
		ctx:       ctx,
	}

	c.pendingCalls.Store(requestID, call)
	defer func() {
		c.pendingCalls.Delete(requestID)
		call.timer.Stop()
	}()

	// 发送请求
	if err := c.publishRequestWithRetry(ctx, req); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to publish request: %v", err)
	}

	if c.config.Debug {
		logx.Infof("vRPC request sent: %s/%s [%s]", serviceName, methodName, requestID)
	}

	// 等待响应
	select {
	case response := <-call.respChan:
		if c.config.Debug {
			logx.Infof("vRPC response received: [%s] status=%d", requestID, response.StatusCode)
		}
		return response, nil
	case <-call.timer.C:
		return nil, status.Errorf(codes.DeadlineExceeded, "request timeout: %s", requestID)
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "request canceled: %v", ctx.Err())
	case <-c.ctx.Done():
		return nil, status.Errorf(codes.Unavailable, "client shutdown")
	}
}

// publishRequestWithRetry 带重试的请求发布
func (c *Client) publishRequestWithRetry(ctx context.Context, req *gen.VRPCRequest) error {
	var lastErr error

	for i := 0; i <= c.config.Retry.Count; i++ {
		if i > 0 {
			select {
			case <-time.After(c.config.Retry.Interval):
			case <-ctx.Done():
				return ctx.Err()
			}

			if c.config.Debug {
				logx.Infof("Retrying request %s, attempt %d", req.RequestId, i)
			}
		}

		if err := c.publisher.Publish(ctx, c.config.Topics.Request, req); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return fmt.Errorf("failed to publish after %d retries: %w", c.config.Retry.Count, lastErr)
}

// 全局客户端管理
var (
	globalClient *Client
	initOnce     sync.Once
	initErr      error
)

// InitClient 初始化全局客户端
func InitClient(config *Config) error {
	initOnce.Do(func() {
		globalClient, initErr = NewClient(config)
		if initErr == nil {
			logx.Info("vRPC client initialized successfully")
		}
	})
	return initErr
}

// MustInitClient 必须成功初始化客户端
func MustInitClient(config ...*Config) {
	var cfg *Config
	if len(config) > 0 {
		cfg = config[0]
	}

	if err := InitClient(cfg); err != nil {
		logx.Errorf("Failed to initialize vRPC client: %v", err)
		panic(err)
	}
}

// GetClient 获取全局客户端
func GetClient() *Client {
	if globalClient == nil {
		panic("vRPC client not initialized, call InitClient first")
	}
	return globalClient
}
