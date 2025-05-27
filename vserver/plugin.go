package vserver

import (
	"context"

	"github.com/nextpkg/vrpc/internal/proto/gen"
)

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// BeforeRequest 请求前处理
	BeforeRequest(ctx context.Context, req *gen.VRPCRequest) error

	// AfterResponse 响应后处理
	AfterResponse(ctx context.Context, req *gen.VRPCRequest, resp *gen.VRPCResponse) error

	// Close 关闭插件
	Close() error
}

// BasePlugin 基础插件实现
type BasePlugin struct {
	name string
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(name string) *BasePlugin {
	return &BasePlugin{name: name}
}

// Name 返回插件名称
func (p *BasePlugin) Name() string {
	return p.name
}

// BeforeRequest 请求前处理（默认实现）
func (p *BasePlugin) BeforeRequest(ctx context.Context, req *gen.VRPCRequest) error {
	return nil
}

// AfterResponse 响应后处理（默认实现）
func (p *BasePlugin) AfterResponse(ctx context.Context, req *gen.VRPCRequest, resp *gen.VRPCResponse) error {
	return nil
}

// Close 关闭插件（默认实现）
func (p *BasePlugin) Close() error {
	return nil
}
