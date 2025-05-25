## What is vRPC
vRPC是一个中间件，通过消息队列将同步RPC调用转换为异步调用。它可以在调度系统和物联网云边缘架构等分布式系统中进行通信，而无需修改服务器端RPC实现。

## Architecture Overview
vRPC中间件在客户端和服务端之间建立了一个异步通信层，主要组件包括：
- 上游代理(Upstream Proxy): 拦截gRPC调用并转换为消息发送到MQ
- 消息队列(MQ): 提供可靠的消息传输
- 下游代理(Downstream Proxy): 接收消息并恢复为RPC调用

## Key component
1. 请求拦截： 
  - 通过vRPC拦截器无缝捕获客户端调用
  - 保持原有RPC接口不变，对客户端透明
2. 消息转换与传输： 
  - 完整保留调用上下文(方法名、元数据、截止时间等)
  - 使用Proto序列化确保高效传输
3. 异步响应机制： 
  - 基于correlationID的请求-响应匹配
  - 使用通道(channel)实现异步等待

## Extended
1. 消息队列适配层： 
  - 设计了通用的消息队列接口
  - 目前实现了Kafka，可轻松扩展到其他MQ
2. RPC框架集成：
  - 针对go-zero框架的集成
  - 通过拦截器模式实现无侵入集成
