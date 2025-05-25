package vrpc

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/nextpkg/vrpc/internal/codec"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Coder 负责编码和解码RPC消息
type Coder struct {
	registry *protoregistry.Types
}

// NewCoder 创建一个新的编码器
func NewCoder() *Coder {
	return &Coder{
		registry: protoregistry.GlobalTypes,
	}
}

// EncodeRequest 将RPC请求编码为消息
func (c *Coder) EncodeRequest(ctx context.Context, id int64, method string, req proto.Message) (*codec.Request, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}

	// 序列化请求
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建请求消息
	request := &codec.Request{
		Base: &codec.Base{
			Id:        id,
			Method:    method,
			Payload:   reqBytes,
			Timestamp: time.Now().UnixNano(),
		},
	}

	return request, nil
}

// DecodeRequest 将消息解码为RPC请求
func (c *Coder) DecodeRequest(ctx context.Context, req *codec.Request) (string, proto.Message, error) {
	if req == nil || req.Base == nil {
		return "", nil, errors.New("request is nil")
	}

	// 获取方法名
	method := req.Base.Method

	// 查找请求类型
	reqType, err := c.findRequestType(method)
	if err != nil {
		return "", nil, fmt.Errorf("failed to find request type: %w", err)
	}

	// 创建请求实例
	reqInstance := reflect.New(reqType).Interface().(proto.Message)

	// 反序列化请求
	if err := proto.Unmarshal(req.Base.Payload, reqInstance); err != nil {
		return "", nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return method, reqInstance, nil
}

// EncodeResponse 将RPC响应编码为消息
func (c *Coder) EncodeResponse(ctx context.Context, id int64, resp proto.Message, err error) (*codec.Response, error) {
	// 创建响应消息
	response := &codec.Response{
		Base: &codec.Base{
			Id:        id,
			Timestamp: time.Now().UnixNano(),
		},
	}

	// 处理错误
	if err != nil {
		response.Error = err.Error()
		return response, nil
	}

	// 序列化响应
	if resp != nil {
		respBytes, err := proto.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		response.Base.Payload = respBytes
	}

	return response, nil
}

// DecodeResponse 将消息解码为RPC响应
func (c *Coder) DecodeResponse(ctx context.Context, resp *codec.Response, method string) (proto.Message, error) {
	if resp == nil || resp.Base == nil {
		return nil, errors.New("response is nil")
	}

	// 处理错误
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	// 查找响应类型
	respType, err := c.findResponseType(method)
	if err != nil {
		return nil, fmt.Errorf("failed to find response type: %w", err)
	}

	// 创建响应实例
	respInstance := reflect.New(respType).Interface().(proto.Message)

	// 反序列化响应
	if err := proto.Unmarshal(resp.Base.Payload, respInstance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return respInstance, nil
}

// findRequestType 查找请求类型
func (c *Coder) findRequestType(method string) (reflect.Type, error) {
	// 这里需要根据方法名查找请求类型
	// 在实际应用中，可能需要使用反射或其他机制来查找
	// 这里简化处理，返回一个通用类型
	return reflect.TypeOf((*proto.Message)(nil)).Elem(), nil
}

// findResponseType 查找响应类型
func (c *Coder) findResponseType(method string) (reflect.Type, error) {
	// 这里需要根据方法名查找响应类型
	// 在实际应用中，可能需要使用反射或其他机制来查找
	// 这里简化处理，返回一个通用类型
	return reflect.TypeOf((*proto.Message)(nil)).Elem(), nil
}
