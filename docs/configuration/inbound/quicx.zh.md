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
  "fec": {
    "enabled": true,
    "max_overhead_percent": 10,
    "max_group_size": 16,
    "max_parity_rows": 1
  },
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

默认使用 `standard`。

#### fec

包级前向纠错（FEC）配置，用于在丢包链路上修复丢失的 QUIC 包，原理、开销数学与实测数据见
[QUICX FEC](../quicx-fec.zh.md)。

**默认开启**，不写这一段就是开启（并使用下面的默认值）。只有客户端与服务端都开启时才生效：
客户端在鉴权请求中声明支持，服务端确认后才真正启用。FEC 是**连接级**协商，协商成功时两端各输出
一条 `QUICX FEC enabled` 的 debug 日志并带上对端地址，因此客户端每次重连都会新增一对，属于正常
现象（详见 [QUICX FEC](../quicx-fec.zh.md#10-日志与观测)）。

运行期间每 10 秒会输出一条统计（窗口内无 FEC 活动时不输出），默认 debug 级别；窗口内实际修复过
包时提升为 info 级别（每连接每分钟最多一条）：

```
QUICX FEC: tx loss 3.4% (peer reported), group 13 rows 2, overhead 15.4% configured / 6.0% measured,
  protected 1200 pkts (195.3 KB), parity 96 pkts (14.2 KB), skipped 2 groups, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 91 pkts, protected 1400 pkts
```

`tx` 一列是本端**发送**方向（丢包率由对端观测后回传），`rx` 一列是本端**接收**方向；两者由不同
端点测量，不要混着看。`skipped` 是因超出上限而**主动放弃保护**的分组数。

#### fec.enabled

是否启用 FEC。默认 `true`；设为 `false` 可单独在本端关闭。

#### fec.max_overhead_percent

冗余流量上限（占被保护流量的百分比）。

默认使用 `10`。该上限约束的是**实际发出的校验字节**：分组太小或包长分布不均时，
FEC 会直接放弃该分组的校验（见统计里的 `skipped`），而不是超发。

#### fec.max_group_size

单个 FEC 分组最多保护的包数量。

默认使用 `32`。分组越大相对开销越低，但丢包修复的等待时间越长。

该值必须大到能在上限内放得下 `max_parity_rows` 行校验，否则多余的行不会被使用
（默认 `32` / `10%` 下 2 行约占 6.8%，可以启用）。

#### fec.max_parity_rows

每个分组最多发送的校验行数。

`2`（默认）每组可修复 2 个丢包（GF(2^8) 上的 Reed-Solomon 校验，类似 RAID 6），
`1` 每组只能修复 1 个丢包（XOR 校验，类似 RAID 5）。

移动链路上的丢包以突发为主，单行校验遇到"同组内 2 个及以上丢包"时整组都修不回来
（实测 15.7 小时日志里 `unrecoverable` 是 `repaired` 的 2～2.5 倍），因此默认使用 2 行。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#入站)。

ALPN 必须为 `h3`。

### QUIC 字段

参阅 [QUIC 字段](/zh/configuration/shared/quic/) 了解详情。

非 QUICX 代理流量的标准 HTTP/3 请求将以优雅 HTTP/3 关闭终止（GOAWAY、排空，再
`H3_NO_ERROR`），等同标准 HTTP/3 服务器行为。
