# Project: AMP Reference Relay (Go)

## 项目概述
这是 AMP (Agent Messaging Protocol) v5.0 的官方 Go 语言参考实现。由 Jason 🍎 实验室开发，Ryan Cooper 负责架构审核。

## 技术栈
- 语言：Go 1.23+
- 编码格式：CBOR (github.com/fxamacker/cbor/v2)
- 核心规范：AMP v5.0 (见 research/lab/LAB_ARCHIVE/)

## 常用命令
- `go build ./...` - 构建项目
- `go test ./...` - 运行单元测试
- `openclaw gateway wake --text "Done" --mode now` - 任务完成通报

## 代码风格 (Ryan Standard)
- **必须**: 使用 CBOR 标签标注所有 Message 字段
- **必须**: 实现 MessageStore 接口以保持存储层抽象
- **必须**: 所有的通信握手必须包含 DID 签名验证
- **禁止**: 在 internal 目录外直接操作存储逻辑

## 关键文件
- `internal/protocol/message.go` - 协议定义
- `internal/storage/store.go` - 存储接口与内存实现
