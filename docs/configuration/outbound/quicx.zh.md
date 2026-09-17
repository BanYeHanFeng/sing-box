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
    "max_overhead_percent": 30,
    "max_group_size": 128,
    "max_parity_rows": 2
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
鉴权请求中声明支持，服务端确认后才真正开启，因此单边开启不会浪费任何带宽。FEC 是**连接级**协商，
协商成功时两端各输出一条 `QUICX FEC enabled` 的 debug 日志并带上对端地址，因此每次重连都会新增
一对，属于正常现象（详见 [QUICX FEC](../quicx-fec.zh.md#5-日志与观测)）。

运行期间每 10 秒会输出一条统计（窗口内无 FEC 活动时不输出），默认 debug 级别；窗口内实际修复过
包时提升为 info 级别（每连接每分钟最多一条）：

```
QUICX FEC: tx loss 3.4% (peer reported), window 128 pkts, rate 5.0% / 4.8% measured,
  protected 1200 pkts (1.4 MB), parity 96 pkts (112.5 KB), skipped 2 rows, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 91 pkts, protected 1400 pkts
```

`tx` 一列是本端**发送**方向（丢包率由对端观测后回传），`rx` 一列是本端**接收**方向；两者由不同
端点测量，不要混着看。`skipped` 是因超出上限而**主动放弃保护**的校验行数。

#### fec.enabled

是否启用 FEC。默认 `true`；设为 `false` 可单独在本端关闭（对端也就不会启用）。

#### fec.max_overhead_percent

冗余流量上限（占被保护流量的百分比）。

默认使用 `30`。该上限约束的是**实际发出的校验字节**：当额度不足以支付一行校验时，FEC 会放弃
这一行（见统计里的 `skipped`），而不是超发。

#### fec.max_group_size

窗口大小（同时保护的包数），默认 `128`。

窗口内同一个包会被约 `窗口大小 × 冗余率` 行校验覆盖，所以窗口越大越能修突发丢包，
代价是内存（约 `窗口大小 × MTU` / 方向）与校验计算量。

#### fec.max_parity_rows

发送端空闲时，为窗口尾部补发的校验行数（默认 `2`）。

一个包在窗口里停留期间被多少行覆盖取决于冗余率，而最后发出的几个包覆盖行数最少，
空闲补尾就是为它们准备的；设为 `1` 可以减少空闲时的校验流量，`2` 能多修一个尾部丢包。

#### fec.baseline_redundancy_percent

保底冗余率（百分比，默认 `5`）。它让干净链路也保持一小部分校验流量，专治"突发到 8% 左右、
之后又突然回落"的链路：首个丢包不必等对端上报（约为 0.5×RTT + 采样时间）才被覆盖。设 `0`
可显式关闭保底冗余。该值仍然受 `max_overhead_percent` 上限约束；`5` 是这类 8% 突发的推荐
膝点，若更在意干净链路上的吞吐可降到 `3`，若突发更严重可升到 `8`。

#### fec.recovered_packet_feedback

把本端用 FEC 修回的包回报给对端（默认 `false`）。

开启后，接收端在修复出包后发送 `FEC_RECOVERED` 帧；发送端把对应的丢包事件喂给拥塞控制，
但不会重传（这些包已经按收到处理并 ACK 过）。这样即使在用 FEC 的链路上，拥塞控制仍能
看到丢包，避免把拥塞当成纯粹的链路损伤。两端都需要理解该帧，所以默认关闭；只有当你希望
FEC 连接参与拥塞控制反馈时才在两端一起打开。

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
