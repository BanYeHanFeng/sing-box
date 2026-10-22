### 结构

```json
{
  "type": "quicx",
  "tag": "quicx-out",

  "server": "127.0.0.1",
  "server_port": 1080,
  "password": "hello",
  "heartbeat": "",
  "bbr_profile": "",
  "qlog_directory": "",
  "qlog_max_size": "",
  "fec": {},
  "network": "",
  "tls": {},

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

quicx 用户密码。

#### heartbeat

发送心跳包以保持连接存活的时间间隔

默认使用 `10s`。

#### bbr_profile

BBR 拥塞控制算法配置，可选 `conservative` `standard` `aggressive`。

默认使用 `conservative`。

#### qlog_directory

文件夹名字，填写后将在文件夹下记录每条 quicx 链接。

默认关闭。

#### qlog_max_size

qlog 目录的总大小上限，单位：MB。

默认使用 `300MB`。

#### fec

UDP 转发的前向纠错配置，默认关闭。

```json
{
  "fec": {
    "enabled": true,
    "block": true,
    "block_size": 10,
    "parity": 2,
    "max_parity": 4,
    "interleave": 4,
    "block_timeout": "10ms"
  }
}
```

`enabled` 需要连接两端同时启用；只有客户端与服务端都支持时才会生效，其余情况保持原有行为。

`block` 启用跨消息块纠删码，关闭时只使用消息内单校验：一条被拆成多个分片的 UDP 消息会额外发送一个 XOR 校验分片，可恢复任意一个丢失分片，且没有额外延迟。

`block_size` 是每个纠删码块包含的消息数量，默认 `10`，上限 `32`。

`parity` 是初始冗余分片数量，默认 `2`。冗余分片会占用拥塞窗口和上行带宽，收到对端的丢包报告后会自动调整。

`max_parity` 是自适应冗余的上限，默认 `4`，上限 `8`。

`interleave` 是块交错深度，默认 `4`。同一个块的消息按该深度分散到不同发送时刻，用于抵抗连续丢包。

`block_timeout` 是块关闭前等待的最长时间，默认 `10ms`。较小的值降低修复时延，较大的值减少尾部浪费。

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
