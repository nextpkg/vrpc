package vrpc

import (
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/nextpkg/vrpc/internal/codec"
)

// Client 处理vRPC通信的客户端
type Client struct {
	upstreamProxy *UpstreamProxy
	correlationID int64
	responses     map[int64]chan *codec.Response
	mu            sync.Mutex
	timeout       time.Duration
}

func NewClient(cfg *Config) *Client {
	// 创建Kafka生产者
	producer := kafka.NewKafkaProducer(&cfg.Request)

	// 创建Kafka消费者
	consumer := kafka.NewKafkaConsumer(&cfg.Respond, func(ctx context.Context, key, value string) error {
		resp := &Response{}
		if err := proto.Unmarshal(msg.Value, resp); err != nil {
			log.Printf("Failed to unmarshal response: %v", err)
			continue
		}

		client.HandleResponse(resp)
	})

	// 创建上游代理
	upstreamProxy := &UpstreamProxy{
		producer: producer,
		consumer: consumer,
	}

	// 创建vRPC客户端
	client := &Client{
		upstreamProxy: upstreamProxy,
		responses:     make(map[int64]chan *codec.Response),
	}

	return &Client{
		upstreamProxy: upstream,
		responses:     make(map[int64]chan *codec.Response),
	}
}

// Call 执行vRPC调用
func (c *Client) Call(ctx context.Context, method string, req, reply any) error {
	// 生成相关ID
	c.mu.Lock()
	corrID := c.correlationID
	c.correlationID++
	c.mu.Unlock()

	// 创建响应通道
	respChan := make(chan *codec.Response, 1)
	c.mu.Lock()
	c.responses[corrID] = respChan
	c.mu.Unlock()

	// 序列化请求
	reqBytes, err := jsoniter.Marshal(req)
	if err != nil {
		return err
	}

	// 创建请求消息
	request := &codec.Request{
		Base: &codec.Base{
			Id:        corrID,
			Method:    method,
			Payload:   reqBytes,
			Timestamp: time.Now().UnixNano(),
		},
	}

	// 发送请求到上游代理
	err = c.upstreamProxy.Send(request)
	if err != nil {
		return err
	}

	// 等待响应
	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.responses, corrID)
		c.mu.Unlock()
		return ctx.Err()
	case resp := <-respChan:
		if resp.Error != "" {
			return errors.New(resp.Error)
		}

		// 反序列化响应
		return jsoniter.Unmarshal(resp.Base.Payload, reply)
	}
}

// HandleResponse 处理来自下游的响应
func (c *Client) HandleResponse(resp *Response) {
	c.mu.Lock()
	ch, ok := c.responses[resp.CorrelationID]
	delete(c.responses, resp.CorrelationID)
	c.mu.Unlock()

	if ok {
		ch <- resp
		close(ch)
	}
}
