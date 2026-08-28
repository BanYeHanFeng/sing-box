### 结构

```json
{
  "type": "quicx",
  "tag": "quicx-in",

  ... // 监听字段

  "users": [
    {
      "name": "sekai",
      "password": "hello"
    }
  ],
  "auth_timeout": "3s",
  "heartbeat": "10s",
  "auth_failure_policy": "h3_close",
  "bbr_profile": "",
  "tls": {
    "enabled": true,
    "certificate_path": "/path/to/certificate.crt",
    "key_path": "/path/to/private.key",
    "alpn": ["h3"]
  },

  ... // QUIC 字段
}
```

### 监听字段

参阅 [监听字段](/zh/configuration/shared/listen/)。

### 字段

#### users

QUICX 用户

#### users.password

==必填==

QUICX 用户密码

由于认证已改为仅使用密码（移除了用户 UUID），每个用户的 `password` 作为客户端用于认证的唯一凭据，因此配置的密码必须唯一。

#### auth_timeout

服务器等待客户端发送认证命令的时间

默认使用 `3s`。

#### heartbeat

发送心跳包以保持连接存活的时间间隔

默认使用 `10s`。

#### auth_failure_policy

服务器处理 quicx 鉴权失败的方式，同时保持传输层与标准 HTTP/3 服务器不可分辨。

| 策略           | 描述                                                       |
|--------------|----------------------------------------------------------|
| `h3_close`   | 以 `H3_NO_ERROR` 关闭 QUIC 连接，等同标准 HTTP/3 正常关闭。 |
| `silent_drop`| 静默丢弃连接、不发送 `CONNECTION_CLOSE`，探测者只能得到超时。                |

默认使用 `h3_close`。

#### bbr_profile

BBR 拥塞控制算法配置，可选 `conservative` `standard` `aggressive`。

留空时为兼容 QUICX 原有行为使用 `conservative`。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#入站)。

ALPN 必须为 `h3`。

### QUIC 字段

参阅 [QUIC 字段](/zh/configuration/shared/quic/) 了解详情。

非 QUICX 代理流量的标准 HTTP/3 请求将以优雅 HTTP/3 关闭终止（GOAWAY、排空，再
`H3_NO_ERROR`），等同标准 HTTP/3 服务器行为。
