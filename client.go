package vrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nextpkg/vrpc/internal/codec"
	"github.com/nextpkg/vrpc/rpcbus"
	"github.com/nextpkg/vrpc/rpcbus/kafka"
	"github.com/nextpkg/vrpc/rpcbus/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

// Client vRPC客户端
type Client struct {
	config    *Config
	conn      *ResponseConnection
	upstream  *UpstreamProxy
	pluginMgr *rpcbus.PluginManager
	producer  rpcbus.MessageProducer
	coder     *codec.Coder
	responses map[string]chan *codec.Response
	respMutex sync.RWMutex
}

// NewClient 创建一个新的vRPC客户端
func NewClient(config *Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// 创建插件管理器
	pluginMgr := rpcbus.NewPluginManager()

	// 注册内置插件
	registerBuiltinPlugins(pluginMgr)

	// 创建编解码器
	coder := codec.NewCoder()

	// 创建客户端
	client := &Client{
		config:    config,
		pluginMgr: pluginMgr,
		coder:     coder,
		responses: make(map[string]chan *codec.Response),
	}

	return client, nil
}

// registerBuiltinPlugins 注册内置插件
func registerBuiltinPlugins(pluginMgr *rpcbus.PluginManager) {
	// 注册Kafka插件
	kafkaPlugin := kafka.NewKafkaPlugin()
	kafkaPlugin.Register(pluginMgr)

	// 注册Kq插件
	kqPlugin := kq.NewKqPlugin()
	kqPlugin.Register(pluginMgr)
}

// RegisterPlugin 注册插件
func (c *Client) RegisterPlugin(plugin rpcbus.Plugin) {
	plugin.Register(c.pluginMgr)
}

// Init 初始化客户端
func (c *Client) Init() error {
	// 初始化插件
	if err := c.pluginMgr.InitPlugin(c.config.PluginName, c.config.PluginConfig); err != nil {
		return fmt.Errorf("failed to init plugin: %w", err)
	}

	// 获取插件
	plugin, err := c.pluginMgr.GetPlugin(c.config.PluginName)
	if err != nil {
		return fmt.Errorf("failed to get plugin: %w", err)
	}

	// 创建生产者
	producer, err := plugin.CreateProducer()
	if err != nil {
		return fmt.Errorf("failed to create producer: %w", err)
	}
	c.producer = producer

	// 创建上游代理
	c.upstream = NewUpstreamProxy(c.producer, c.config.RequestTopic)

	// 创建响应连接
	consumer, err := plugin.CreateConsumer()
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	// 创建响应连接的生产者
	respProducer, err := plugin.CreateProducer()
	if err != nil {
		return fmt.Errorf("failed to create response producer: %w", err)
	}

	c.conn = NewResponseConnection(respProducer, c.config.ResponseTopic)

	// 启动响应处理
	go c.handleResponses()

	return nil
}

// Call 调用远程方法
func (c *Client) Call(ctx context.Context, serviceName, methodName string, request interface{}, response interface{}) error {
	// 创建请求ID
	requestID := generateRequestID()

	// 创建响应通道
	respCh := make(chan *codec.Response, 1)
	c.respMutex.Lock()
	c.responses[requestID] = respCh
	c.respMutex.Unlock()

	// 清理函数
	defer func() {
		c.respMutex.Lock()
		delete(c.responses, requestID)
		c.respMutex.Unlock()
		close(respCh)
	}()

	// 编码请求数据
	reqData, err := c.coder.EncodeRequest(request)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	// 创建请求
	req := &codec.Request{
		Base: &codec.Base{
			Id:        requestID,
			Service:   serviceName,
			Method:    methodName,
			Timestamp: time.Now().UnixNano(),
			Payload:   reqData,
		},
	}

	// 发送请求
	if err := c.upstream.Send(ctx, req); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// 等待响应或超时
	select {
	case resp := <-respCh:
		if resp.Error != "" {
			return errors.New(resp.Error)
		}
		return c.coder.DecodeResponse(resp.Base.Payload, response)
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.config.Timeout):
		return errors.New("rpc call timeout")
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	var errs []error

	// 关闭响应连接
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close response connection: %w", err))
		}
	}

	// 关闭生产者
	if c.producer != nil {
		if err := c.producer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close producer: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing client: %v", errs)
	}

	return nil
}

// handleResponses 处理响应
func (c *Client) handleResponses() {
	err := c.conn.Consume(context.Background(), func(response *codec.Response) error {
		if response.Base == nil {
			logx.Error("Received response with nil base")
			return nil
		}

		c.respMutex.RLock()
		respCh, ok := c.responses[response.Base.Id]
		c.respMutex.RUnlock()

		if !ok {
			logx.Infof("Response for unknown request ID: %s", response.Base.Id)
			return nil
		}

		respCh <- response
		return nil
	})

	if err != nil {
		logx.Errorf("Error consuming responses: %v", err)
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
