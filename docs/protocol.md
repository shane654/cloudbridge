# CloudBridge 协议设计

## 帧协议

所有隧道传输（WebRTC DataChannel / QUIC / Relay TCP）使用统一的帧协议进行多路复用。

### 帧格式

```
+----------+--------+-----------+----------+
| StreamID | Type   | Length    | Payload  |
| 2 bytes  | 1 byte | 4 bytes   | N bytes  |
+----------+--------+-----------+----------+
```

### 流类型 (StreamID)

| ID | 名称 | 说明 |
|----|------|------|
| 0x0000 | Control | 控制流（会话管理、窗口调整） |
| 0x0001 | SSH | SSH 协议流 |
| 0x0002 | Shell | 交互式终端流 |
| 0x0003 | RDP | 远程桌面流 |
| 0x0004 | Docker | Docker 管理流 |
| 0x0005 | VNC | VNC 协议流 |

### 帧类型 (Type)

| 值 | 名称 | 说明 |
|----|------|------|
| 0x01 | DATA | 数据帧 |
| 0x02 | OPEN_STREAM | 打开新流 |
| 0x03 | CLOSE_STREAM | 关闭流 |
| 0x04 | WINDOW_UPDATE | 流控窗口更新 |
| 0x05 | PING | 保活探测 |
| 0x06 | PONG | 保活应答 |

### 限制

- 最大帧载荷：64 KiB
- 帧头大小：7 字节

## 信令协议

信令通过 WebSocket 传输，使用 JSON 编码。

### 消息格式

```json
{
  "type": "message_type",
  "data": { ... }
}
```

### 消息类型

#### 设备管理

| 类型 | 方向 | 说明 |
|------|------|------|
| `register` | Agent → Server | 设备注册 |
| `register_ack` | Server → Agent | 注册确认（含 Token） |
| `heartbeat` | Agent → Server | 心跳（30s 间隔） |
| `heartbeat_ack` | Server → Agent | 心跳确认 |

#### 连接管理

| 类型 | 方向 | 说明 |
|------|------|------|
| `connect_request` | App → Server → Agent | 请求连接 |
| `connect_response` | Agent → Server → App | 连接应答 |
| `disconnect` | 双向 | 断开连接 |

#### WebRTC 信令

| 类型 | 方向 | 说明 |
|------|------|------|
| `sdp_offer` | 双向 | SDP Offer |
| `sdp_answer` | 双向 | SDP Answer |
| `ice_candidate` | 双向 | ICE 候选 |

#### 传输协商

| 类型 | 方向 | 说明 |
|------|------|------|
| `transport_negotiate` | 双向 | 传输模式协商 |
| `transport_ready` | 双向 | 传输就绪确认 |

#### 错误

| 类型 | 方向 | 说明 |
|------|------|------|
| `error` | Server → 客户端 | 错误信息 |

## 连接流程

```
App                         Signal Server                   Agent
 |                               |                            |
 | 1. WebSocket Connect          |                            |
 |------------------------------►|                            |
 |                               | 2. WebSocket Connect       |
 |                               |◄---------------------------|
 |                               |                            |
 |                               | 3. Register                |
 |                               |◄---------------------------|
 |                               | 4. Register Ack (Token)    |
 |                               |--------------------------►|
 |                               |                            |
 | 5. Connect Request            |                            |
 |------------------------------►| 6. Forward Connect Request |
 |                               |--------------------------►|
 |                               |                            |
 |                               | 7. Connect Response (accept)|
 |                               |◄---------------------------|
 | 8. Forward Connect Response   |                            |
 |◄------------------------------|                            |
 |                               |                            |
 | 9. SDP Offer / ICE Candidates |                            |
 |───────────────────────────────────(P2P negotiation)────────►|
 |                               |                            |
 | 10. P2P DataChannel (direct)  |                            |
 |◄──────────────────────────────(or via Relay)──────────────►|
```

## 加密

- **WebRTC DataChannel**：内置 DTLS 加密
- **QUIC**：内置 TLS 1.3
- **Relay TCP**：TLS 1.3 + 应用层 E2E 加密（Agent ↔ App 双端加密，中继无法解密）