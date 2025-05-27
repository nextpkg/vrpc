package vrpc

import (
	"sync"

	"github.com/nextpkg/vrpc/vclient"
	"google.golang.org/grpc"
)

var (
	client     *vclient.Client
	clientOnce sync.Once
	clientMu   sync.RWMutex
)

// MustInitClient 初始化vRPC客户端（使用默认配置）
func MustInitClient() {
	clientOnce.Do(func() {
		c, err := vclient.NewClient(nil)
		if err != nil {
			panic(err)
		}
		client = c
		vclient.SetGlobalClient(c)
	})
}

// MustInitClientWithConfig 使用指定配置初始化vRPC客户端
func MustInitClientWithConfig(config *vclient.Config) {
	clientMu.Lock()
	defer clientMu.Unlock()

	c, err := vclient.NewClient(config)
	if err != nil {
		panic(err)
	}
	client = c
	vclient.SetGlobalClient(c)
}

// InitClient 初始化vRPC客户端
func InitClient(config *vclient.Config) error {
	clientMu.Lock()
	defer clientMu.Unlock()

	c, err := vclient.NewClient(config)
	if err != nil {
		return err
	}
	client = c
	vclient.SetGlobalClient(c)
	return nil
}

// GetClient 获取vRPC客户端实例
func GetClient() *vclient.Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	return client
}

// Interceptor 返回vRPC拦截器
func Interceptor() grpc.UnaryClientInterceptor {
	if client == nil {
		MustInitClient()
	}
	return client.UnaryInterceptor()
}

// Close 关闭vRPC客户端
func Close() error {
	clientMu.Lock()
	defer clientMu.Unlock()

	if client != nil {
		err := client.Close()
		client = nil
		return err
	}
	return nil
}
