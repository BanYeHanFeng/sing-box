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
  "fec": {},
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

#### fec

前向纠错，保护本端发出的包。

客户端发出的是请求，服务端发出的是应答，所以这一项只覆盖请求方向；两个方向都要保护时，入站与出站都要配置。冗余包与数据包一样占拥塞窗口，因此它不会让链路更拥塞，只会把一部分带宽换成无需重传的恢复。

```json
{
  "fec": {
    "data_shards": 10,
    "parity_shards": 2
  }
}
```

#### fec.data_shards

==必填==

一组包含的包数，`2` 到 `32`。

#### fec.parity_shards

一组附带的冗余包数，`0` 到 `8`，且不超过 `data_shards`。

默认为 `0`，此时按 `2` 处理。

#### network

启用的网络协议。

`tcp` 或 `udp`。

默认所有。

### 拨号字段

参阅 [拨号字段](/zh/configuration/shared/dial/) 了解详情。

### 说明

quicx 不是 QUIC：它运行在自己的 UDP 传输层上，不使用 TLS 与 HTTP/3，因此没有证书、ALPN、qlog 与 QUIC 相关配置项。
