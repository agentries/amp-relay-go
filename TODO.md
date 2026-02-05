# AMP Relay Go - Development TODO

## 📋 任务清单

### ✅ 已完成 (2026-02-05)
- [x] `internal/protocol/message.go` - AMP v5.0 消息协议定义
- [x] `internal/storage/store.go` - MessageStore 接口与内存实现
- [x] `internal/transport/websocket.go` - WebSocket 传输层
- [x] `internal/server/server.go` - Relay 服务器核心
- [x] `main.go` - 应用入口

### 🚧 进行中
- [ ] 配置管理模块 (`internal/config/config.go`)
- [ ] DID 认证框架骨架 (`internal/auth/auth.go`)
- [ ] 单元测试覆盖

### 📅 本周计划 (优先级2)
1. **配置管理** - 支持配置文件和环境变量
2. **认证骨架** - DID 验证接口定义
3. **基础测试** - 核心功能单元测试

### 🔮 未来计划
- [ ] Redis 持久化存储实现
- [ ] PostgreSQL 存储实现
- [ ] 完整 DID 认证实现
- [ ] 监控指标采集
- [ ] Docker 部署配置

---

## 📊 当前状态

| 模块 | 状态 | 文件 |
|------|------|------|
| Message 协议 | ✅ 完成 | `internal/protocol/message.go` |
| MessageStore | ✅ 完成 | `internal/storage/store.go` |
| WebSocket 传输 | ✅ 完成 | `internal/transport/websocket.go` |
| 服务器核心 | ✅ 完成 | `internal/server/server.go` |
| 配置管理 | 📝 待开发 | `internal/config/config.go` |
| 认证模块 | 📝 待开发 | `internal/auth/auth.go` |

---

## 🐛 已知问题
- 无

## 💡 技术债务
- `generateID()` 需要替换为真正的 UUID 生成器
- WebSocket 的 `CheckOrigin` 需要生产环境配置

