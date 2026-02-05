# AMP Relay Go - Development TODO

## 📋 任务清单

### ✅ 已完成 (2026-02-05)
- [x] `internal/protocol/message.go` - AMP v5.0 消息协议定义
- [x] `internal/storage/store.go` - MessageStore 接口与内存实现
- [x] `internal/transport/websocket.go` - WebSocket 传输层 (~300行)
- [x] `internal/server/server.go` - Relay 服务器核心 (~400行)
- [x] `internal/config/config.go` - 配置管理模块 (YAML/JSON/env)
- [x] `internal/auth/auth.go` - DID 认证框架骨架
- [x] 单元测试覆盖 - 全部通过

### 🐛 代码审查发现的问题 (待修复)
1. **Bug #1**: `randomString()` 和 `getRandomString()` 函数不是真正随机的
   - 位置: `websocket.go:267`, `message.go:83`
   - 问题: 只是循环使用charset的前几个字符，不是随机选择
   - 修复: 使用 `crypto/rand` 生成真正的随机字符串

2. **Bug #2**: `handleHealth()` 函数中的客户端计数转换错误
   - 位置: `websocket.go:186`
   - 问题: `string(rune(ws.GetClientCount()+48))` 只适用于个位数
   - 修复: 使用 `fmt.Sprintf()` 或 `strconv.Itoa()`

### 🚧 进行中
- [ ] 修复代码审查发现的2个bug

### 📅 明日计划 (2026-02-06)
1. **修复 Bug #1** - randomString 真正随机化
2. **修复 Bug #2** - handleHealth 客户端计数修复
3. **实现 Ryan 的 Authenticator 接口** - 根据 Ryan 提供的接口定义调整
4. **安排同步会议** - auth 完成后与 Ryan 同步

### 🔮 未来计划
- [ ] Redis 持久化存储实现
- [ ] PostgreSQL 存储实现
- [ ] 完整 DID 认证实现 (Agentries 集成)
- [ ] 监控指标采集
- [ ] Docker 部署配置

---

## 📊 当前状态

| 模块 | 状态 | 文件 | 行数 |
|------|------|------|------|
| Message 协议 | ✅ 完成 | `internal/protocol/message.go` | ~100 |
| MessageStore | ✅ 完成 | `internal/storage/store.go` | ~100 |
| WebSocket 传输 | ✅ 完成 | `internal/transport/websocket.go` | ~300 |
| 服务器核心 | ✅ 完成 | `internal/server/server.go` | ~400 |
| 配置管理 | ✅ 完成 | `internal/config/config.go` | ~400 |
| 认证模块 | ✅ 骨架 | `internal/auth/auth.go` | ~350 |
| **总计** | ✅ | | **~1650** |

---

## 📝 技术债务
- `generateID()` 需要替换为真正的 UUID 生成器
- WebSocket 的 `CheckOrigin` 需要生产环境配置
- 需要添加 auth 中间件集成到 server

## 🎯 本周目标
- W4 E2E demo: Ryan ↔ Jason 消息交换
- 当前进度: 70% (核心功能完成，待修复bug和集成auth)
