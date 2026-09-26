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
  "qlog_directory": "",
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

#### auth_timeout

How long the server should wait for the client to send the authentication command

`3s` is used by default.

#### heartbeat

Interval for sending heartbeat packets for keeping the connection alive

`10s` is used by default.

#### auth_failure_policy

How the server handles quicx authentication failures and standard HTTP/3 requests that are not QUICX traffic, while keeping the transport layer indistinguishable from a standard HTTP/3 server.

| Policy       | Description                                                                                     |
|--------------|-------------------------------------------------------------------------------------------------|
| `h3_close`   | Close the QUIC connection with `H3_NO_ERROR`, like a normal HTTP/3 connection close. |
| `silent_drop`| Silently drop the connection without sending `CONNECTION_CLOSE`; the prober only sees a timeout. The connection is only kept for a grace period (30 seconds by default) and is reclaimed locally afterwards, so a peer sending keepalives cannot pin its resources.|

`h3_close` is used by default.

#### bbr_profile

BBR congestion control algorithm profile, one of `conservative` `standard` `aggressive`.

`conservative` is used by default.

#### qlog_directory

Write a [qlog](https://datatracker.ietf.org/doc/draft-ietf-quic-qlog-main-schema/)
trace for every QUIC connection into this directory, named
`<connection id>_server.sqlog`.

Disabled by default. The directory is created at startup, and a path that
cannot be written makes sing-box fail to start instead of tracing nothing.

!!! note ""
    Every connection gets its own file and nothing prunes the directory, so it
    only grows. Enable this while debugging a specific problem, not for
    permanent operation.

The traces can be opened with [qvis](https://qvis.quictools.info/).

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

The ALPN must be `h3`.

### 0-RTT and replay

The server accepts 0-RTT data so that the transport layer stays
indistinguishable from a standard HTTP/3 server. As QUIC 0-RTT has no replay
protection of its own, the client attaches a random, per-connection nonce to
its authentication request and the server remembers the nonces of authenticated
sessions:

- Another session presenting the same nonce is treated as a replayed 0-RTT
  flight: it is terminated following `auth_failure_policy`, so no connection to
  the destination is established.
- A session presenting its own nonce again (a client racing two authentication
  streams, or resending its authentication after a rejected 0-RTT attempt) is
  not a replay.

The server remembers the nonces of the last 65536 authenticated sessions for at
most 24 hours, after which a replay is no longer guaranteed to be rejected. The
state lives in the process: a deployment with several instances, or a restart,
resets the window.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

Standard HTTP/3 requests that are not QUICX proxy traffic do not establish a
proxy connection; they are terminated following `auth_failure_policy`: closed
with `H3_NO_ERROR` for `h3_close`, or silently dropped for `silent_drop` (no
close frame is sent during the grace period).
