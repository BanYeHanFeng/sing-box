### Structure

```json
{
  "type": "quicx",
  "tag": "quicx-in",

  ... // Listen Fields

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

  ... // QUIC Fields

  "masquerade": "" // or {}
}
```

### Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

### Fields

#### users

QUICX users

#### users.uuid

==Required==

QUICX user uuid

#### users.password

QUICX user password

#### auth_timeout

How long the server should wait for the client to send the authentication command

`3s` is used by default.

#### heartbeat

Interval for sending heartbeat packets for keeping the connection alive

`10s` is used by default.

#### auth_failure_policy

How the server handles quicx authentication failures while keeping the transport layer indistinguishable from a standard HTTP/3 server.

| Policy       | Description                                                                                     |
|--------------|-------------------------------------------------------------------------------------------------|
| `h3_close`   | Gracefully close the connection: send a GOAWAY frame, drain in-flight requests, then close the QUIC connection with `H3_NO_ERROR`, like a normal HTTP/3 server shutdown. |
| `silent_drop`| Silently drop the connection without sending `CONNECTION_CLOSE`; the prober only sees a timeout.|

`h3_close` is used by default.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

The ALPN must be `h3`.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

#### masquerade

HTTP3 server behavior (URL string configuration) for standard HTTP/3 connections.

| Scheme       | Example                 | Description        |
|--------------|-------------------------|--------------------|
| `file`       | `file:///var/www`       | As a file server   |
| `http/https` | `http://127.0.0.1:8080` | As a reverse proxy |

Conflict with `masquerade.type`.

A 404 page will be returned if masquerade is not configured.

#### masquerade.type

HTTP3 server behavior (Object configuration) for standard HTTP/3 connections.

| Type     | Description                 | Fields                              |
|----------|-----------------------------|-------------------------------------|
| `file`   | As a file server            | `directory`                         |
| `proxy`  | As a reverse proxy          | `url`, `rewrite_host`               |
| `string` | Reply with a fixed response | `status_code`, `headers`, `content` |

Conflict with `masquerade`.

A 404 page will be returned if masquerade is not configured.

#### masquerade.directory

File server root directory.

#### masquerade.url

Reverse proxy target URL.

#### masquerade.rewrite_host

Rewrite the `Host` header to the target URL.

#### masquerade.status_code

Fixed response status code.

#### masquerade.headers

Fixed response headers.

#### masquerade.content

Fixed response content.
