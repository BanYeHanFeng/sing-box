### Structure

```json
{
  "type": "quicx",
  "tag": "quicx-in",

  ... // Listen Fields

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
  "tls": {
    "enabled": true,
    "certificate_path": "/path/to/certificate.crt",
    "key_path": "/path/to/private.key",
    "alpn": ["h3"]
  },

  ... // QUIC Fields
}
```

### Listen Fields

See [Listen Fields](/configuration/shared/listen/) for details.

### Fields

#### users

QUICX users

#### users.password

==Required==

QUICX user password

Now that authentication is password-only (the user UUID has been removed), each
user's `password` acts as the unique credential that the client uses to
authenticate, so passwords must be unique among the configured users.

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
| `h3_close`   | Close the QUIC connection with `H3_NO_ERROR`, like a normal HTTP/3 connection close. |
| `silent_drop`| Silently drop the connection without sending `CONNECTION_CLOSE`; the prober only sees a timeout.|

`h3_close` is used by default.

#### bbr_profile

BBR congestion control algorithm profile, one of `conservative` `standard` `aggressive`.

If empty, `conservative` is used for compatibility with the historical QUICX behavior.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

The ALPN must be `h3`.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

Standard HTTP/3 requests that are not QUICX proxy traffic are terminated with a
graceful HTTP/3 shutdown (GOAWAY, drain, then `H3_NO_ERROR`), matching the
behavior of a normal HTTP/3 server.
