package vrpc

import "fmt"

// Request 请求消息结构
type Request struct {
	CorrelationID uint64
	Method        string
	Payload       []byte
}

// Response 响应消息结构
type Response struct {
	CorrelationID uint64
	Error         string
	Payload       []byte
}

// ResponseConnection 负责将响应发送回客户端
type ResponseConnection struct {
	producer MessageProducer
	topic    string
}

// Send 发送响应到响应队列
func (r *ResponseConnection) Send(resp *codec.Message) error {
	// 序列化响应
	respBytes, err := proto.Marshal(resp)
	if err != nil {
		return err
	}

	// 发送消息到MQ
	return r.producer.Produce(r.topic, fmt.Sprintf("%d", resp.Id), respBytes)
}
