# CloudBridge 云桥

通过手机连接远程 Windows/Linux 设备的开源平台。支持直连和桥接，解决家庭宽带无公网 IP 的核心痛点。

## 架构

```
┌──────────────┐         ┌─────────────────────────────────┐         ┌──────────────┐
│              │  WS控制  │      CloudBridge Server (Go)     │  WS控制  │              │
│  Flutter App │◄────────►│                                  │◄────────►│    Agent     │
│  (手机端)     │  信令通道  │  ┌─────────┐ ┌──────┐ ┌──────┐  │         │ (Win/Linux)  │
│              │         │  │ Signal   │ │ STUN │ │Relay │  │         │              │
└──────┬───────┘         │  │ Server   │ │Server│ │Server│  │         └──────┬───────┘
       │                 │  └─────────┘ └──────┘ └──────┘  │                │
       │                 └─────────────────┬─────────────────┘                │
       │                                   │                                 │
       │        P2P 直连 (WebRTC/QUIC)      │    Relay 兜底转发               │
       │◄──────────────────────────────────►│◄───────────────────────────────►│
       │        (STUN/ICE 探测后建立)        │    (P2P失败时自动回落)            │
```

**连接优先级**：P2P 直连（WebRTC DataChannel / QUIC）→ Relay 中继（自动回落）

## 快速开始

### 编译

```bash
# 安装依赖
go mod tidy

# 编译
make all

# 或分别编译
make server  # → bin/cloudbridge-server
make agent   # → bin/cloudbridge-agent
```

### Docker 部署

```bash
# 一键启动 Server + Agent
cd deploy
docker compose up -d
```

### 手动运行

```bash
# 启动服务器（Signal + STUN + Relay）
make run-server

# 启动 Agent（在另一终端）
make run-agent
```

## 核心功能

| 功能 | 状态 | 说明 |
|------|------|------|
| WebSocket 信令 | ✅ | 设备注册、心跳、连接协商 |
| STUN 服务器 | ✅ | NAT 类型探测 (RFC 5389) |
| TCP Relay 中继 | ✅ | P2P 失败时自动回落 |
| Shell 代理 | ✅ | 交互式 PTY 终端 |
| 帧协议多路复用 | ✅ | 单连接多会话 |
| WebRTC P2P | 🚧 | Phase 2 |
| SSH 代理 | 🚧 | Phase 3 |
| RDP 代理 | 🚧 | Phase 3 |
| Docker 管理 | 🚧 | Phase 3 |
| Flutter App | 🚧 | Phase 1 (MVP) |
| E2E 加密 | 📋 | Phase 4 |

## 项目结构

```
CloudBridge/
├── cmd/
│   ├── server/          # 服务器入口
│   └── agent/           # Agent 入口
├── internal/
│   ├── signal/          # WebSocket 信令服务
│   ├── stun/            # STUN 服务器 (RFC 5389)
│   ├── relay/           # TCP Relay 中继服务
│   ├── protocol/        # 帧协议 & 信令消息
│   ├── agent/           # Agent 逻辑
│   │   └── proxy/       # 协议代理 (Shell/SSH/RDP)
│   └── api/             # REST API (预留)
├── pkg/
│   ├── crypto/          # 密钥 & E2E 加密 (预留)
│   └── models/          # 数据模型 (预留)
├── app/                 # Flutter 应用 (待实现)
├── deploy/              # Docker 部署配置
└── docs/                # 文档
```

## 技术栈

- **服务端**：Go 1.22+
- **Agent**：Go（单二进制，跨平台）
- **移动端**：Flutter（待实现）
- **信令**：WebSocket (gorilla/websocket)
- **穿透**：STUN/ICE + WebRTC DataChannel + QUIC（待实现）
- **中继**：TCP Relay（当前）/ TURN（待实现）

## 应用场景

1. **Vibe Coding 远程开发**：手机连接家用电脑上的 Claude Code / Cursor
2. **家用云管理**：远程管理 Proxmox/Unraid/Docker 容器
3. **移动运维**：SSH/RDP 远程故障排查
4. **内网穿透**：替代 frp/ngrok 的零配置方案

## License

MIT