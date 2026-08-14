### 结构

```json
{
  "type": "quicx",
  "tag": "quicx-in",

  ... // 监听字段

  "users": [
    {
      "name": "sekai",
      "uuid": "059032A9-7D40-4A96-9BB1-36823D848068",
      "password": "hello"
    }
  ],
  "auth_timeout": "3s",
  "heartbeat": "10s",
  "auth_failure_policy": "h3_close",
  "tls": {
    "enabled": true,
    "certificate_path": "/path/to/certificate.crt",
    "key_path": "/path/to/private.key",
    "alpn": ["h3"]
  },

  ... // QUIC 字段

  "masquerade": "" // 或 {}
}
```

### 监听字段

参阅 [监听字段](/zh/configuration/shared/listen/)。

### 字段

#### users

QUICX 用户

#### users.uuid

==必填==

QUICX 用户 UUID

#### users.password

QUICX 用户密码

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
| `h3_close`   | 优雅关闭连接：发送 GOAWAY 帧、排空在途请求，再以 `H3_NO_ERROR` 关闭 QUIC 连接，等同标准 HTTP/3 服务器关闭。 |
| `silent_drop`| 静默丢弃连接、不发送 `CONNECTION_CLOSE`，探测者只能得到超时。                |

默认使用 `h3_close`。

#### tls

==必填==

TLS 配置，参阅 [TLS](/zh/configuration/shared/tls/#入站)。

ALPN 必须为 `h3`。

### QUIC 字段

参阅 [QUIC 字段](/zh/configuration/shared/quic/) 了解详情。

#### masquerade

标准 HTTP/3 连接时的 HTTP3 服务器行为（URL 字符串配置）。

| 协议         | 示例                     | 描述        |
|------------|-------------------------|-----------|
| `file`     | `file:///var/www`       | 作为文件服务器  |
| `http/https` | `http://127.0.0.1:8080` | 作为反向代理   |

与 `masquerade.type` 冲突。

如果未配置 masquerade，将返回 404 页面。

#### masquerade.type

标准 HTTP/3 连接时的 HTTP3 服务器行为（对象配置）。

| 类型     | 描述         | 字段                                  |
|--------|------------|-------------------------------------|
| `file` | 作为文件服务器   | `directory`                         |
| `proxy`| 作为反向代理    | `url`, `rewrite_host`               |
| `string`| 回复固定响应   | `status_code`, `headers`, `content` |

与 `masquerade` 冲突。

如果未配置 masquerade，将返回 404 页面。

#### masquerade.directory

文件服务器根目录。

#### masquerade.url

反向代理目标 URL。

#### masquerade.rewrite_host

将 `Host` 头重写为目标 URL。

#### masquerade.status_code

固定响应状态码。

#### masquerade.headers

固定响应头。

#### masquerade.content

固定响应内容。
