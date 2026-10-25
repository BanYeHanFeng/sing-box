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
  "udp_timeout": "",
  "idle_timeout": "",
  "heartbeat": "",
  "bbr_profile": "",
  "fec": {}
}
```

### 监听字段

参阅 [监听字段](/zh/configuration/shared/listen/)。

### 字段

#### users

quicx 用户。

#### users.password

==必填==

quicx 用户密码。

密码不上线：握手时两端各自用密码派生的密钥证明知道对方，服务端据此找到用户。

#### udp_timeout

NAT 超时时间

默认使用 `5m`。

#### idle_timeout

连接空闲超时时间

默认使用 `30s`。

#### heartbeat

发送心跳包以保持连接存活的时间间隔

默认使用 `10s`。

#### bbr_profile

BBR 拥塞控制算法配置，可选 `conservative` `standard` `aggressive`。

默认使用 `conservative`。

#### fec

前向纠错，保护本端发出的包。

服务端发出的是应答，客户端发出的是请求，所以这一项只覆盖应答方向；两个方向都要保护时，入站与出站都要配置。冗余包与数据包一样占拥塞窗口，因此它不会让链路更拥塞，只会把一部分带宽换成无需重传的恢复。

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

一组附带的冗余包数，`0` 到 `data_shards`。

默认为 `0`，由丢包率决定：一组拿到它预期丢失数量的 1.5 倍的冗余，路径没有丢包时不发冗余。

### 说明

quicx 不是 QUIC：它运行在自己的 UDP 传输层上，不使用 TLS 与 HTTP/3，也没有对应的配置项。

握手是一来一回两个包，用密码与 X25519 派生的密钥认证并加密，之后的一切（包括 UDP 转发）都在同一个加密通道里，因此每个连接不重复握手，也不需要证书。
