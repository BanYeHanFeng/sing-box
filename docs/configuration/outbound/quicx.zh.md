### 结构

```json
{
  "type": "quicx",
  "tag": "quicx-out",

  "server": "127.0.0.1",
  "server_port": 443,
  "password": "hello",
  "heartbeat": "10s",
  "bbr_profile": "",
  "network": "tcp",
  "tls": {
    "enabled": true,
    "server_name": "example.com",
    "alpn": ["h3"]
  },

  ... // QUIC 字段

  ... // 拨号字段
}
```

### 字段

#### server

==必填==

服务器地址。

#### server_port

==必填==

服务器端口。

#### password

==必填==

QUICX 用户密码

该密码是用于向服务器认证的仅密码凭据（用户 UUID 已被移除）。

#### heartbeat

发送心跳包以保持连接存活的时间间隔

默认使用 `10s`。

#### bbr_profile

BBR 拥塞控制算法配置，可选 `conservative` `standard` `aggressive`。

留空时为兼容 QUICX 原有行为使用 `conservative`。

#### network

启用的网络协议。

`tcp` 或 `udp`。

默认所有。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#出站)。

ALPN 必须为 `h3`。

### QUIC 字段

参阅 [QUIC 字段](/zh/configuration/shared/quic/) 了解详情。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/) 了解详情。
