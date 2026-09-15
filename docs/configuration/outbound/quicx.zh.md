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
  "fec": {
    "enabled": true,
    "max_overhead_percent": 10,
    "max_group_size": 16,
    "max_parity_rows": 1
  },
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

默认使用 `standard`。

#### fec

包级前向纠错（FEC）配置，用于在丢包链路上修复丢失的 QUIC 包，原理、开销数学与实测数据见
[QUICX FEC](../quicx-fec.zh.md)。

**默认开启**，不写这一段就是开启（并使用下面的默认值）。只有在服务端也开启时才会生效：客户端在
鉴权请求中声明支持，服务端确认后才真正开启，因此单边开启不会浪费任何带宽。

运行期间每 10 秒会输出一条统计（窗口内无 FEC 活动时不输出），默认 debug 级别；窗口内实际修复过
包时提升为 info 级别（每连接每分钟最多一条）：

```
QUICX FEC: path loss 3.4%, group 13 (overhead 7.7%), repaired 128, unrecoverable 9,
  parity 96 sent / 91 received, protected 1200 packets (14.2 KB parity data)
```

#### fec.enabled

是否启用 FEC。默认 `true`；设为 `false` 可单独在本端关闭（对端也就不会启用）。

#### fec.max_overhead_percent

冗余流量上限（占被保护流量的百分比）。

默认使用 `10`。无论链路丢包多严重，FEC 都不会超过该上限。

#### fec.max_group_size

单个 FEC 分组最多保护的包数量。

默认使用 `16`。分组越大相对开销越低，但丢包修复的等待时间越长。

#### fec.max_parity_rows

每个分组最多发送的校验行数。

`1`（默认）每组可修复 1 个丢包（XOR 校验，类似 RAID 5），`2` 每组可修复 2 个丢包（GF(2^8) 上的 Reed-Solomon 校验，类似 RAID 6）。

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
