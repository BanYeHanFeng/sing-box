### 结构

```json
{
  "type": "quicx",
  "tag": "quicx-out",

  "server": "127.0.0.1",
  "server_port": 1080,
  "password": "hello",
  "idle_timeout": "",
  "heartbeat": "",
  "bbr_profile": "",
  "network": "",

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

quicx 用户密码。

#### idle_timeout

连接空闲超时时间

默认使用 `30s`。

#### heartbeat

发送心跳包以保持连接存活的时间间隔

默认使用 `10s`。

#### bbr_profile

BBR 拥塞控制算法配置，可选 `conservative` `standard` `aggressive`。

默认使用 `conservative`。

#### network

启用的网络协议。

`tcp` 或 `udp`。

默认所有。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/) 了解详情。

### 说明

quicx 不是 QUIC：它运行在自己的 UDP 传输层上，不使用 TLS 与 HTTP/3，因此没有证书、ALPN、qlog 与 QUIC 相关配置项。
