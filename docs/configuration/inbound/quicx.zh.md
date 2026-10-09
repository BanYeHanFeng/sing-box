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
  "auth_timeout": "",
  "heartbeat": "",
  "auth_failure_policy": "",
  "bbr_profile": "",
  "qlog_directory": "",
  "qlog_max_size": "",
  "tls": {},

  ... // QUIC 字段
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

#### auth_timeout

服务器等待客户端发送认证命令的时间

默认使用 `3s`。

#### heartbeat

发送心跳包以保持连接存活的时间间隔

默认使用 `10s`。

#### auth_failure_policy

服务器处理 quicx 鉴权失败以及非 quicx 标准 HTTP/3 请求的方式，同时保持传输层与标准 HTTP/3 服务器不可分辨。

| 策略 | 描述 |
| - | - |
| `h3_close` | 关闭 quic 连接，等同标准 HTTP/3 正常关闭。 |
| `silent_drop`| 静默丢弃连接，服务端仅在宽限期内保留该连接，之后本地回收。 |

默认使用 `h3_close`。

#### bbr_profile

BBR 拥塞控制算法配置，可选 `conservative` `standard` `aggressive`。

默认使用 `conservative`。

#### qlog_directory

文件夹名字，填写后将在文件夹下记录每条 quicx 链接。

默认关闭。

#### qlog_max_size

qlog 目录的总大小上限，单位：MB。

默认使用 `300MB`。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#入站)。

ALPN 必须为 `h3`。


### QUIC 字段

参阅 [QUIC 字段](/zh/configuration/shared/quic/) 了解详情。

