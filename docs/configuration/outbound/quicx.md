### Structure

```json
{
  "type": "quicx",
  "tag": "quicx-out",

  "server": "127.0.0.1",
  "server_port": 443,
  "password": "hello",
  "heartbeat": "10s",
  "bbr_profile": "",
  "network": "tcp",
  "tls": {
    "enabled": true,
    "server_name": "example.com",
    "alpn": ["h3"]
  },

  ... // QUIC Fields

  ... // Dial Fields
}
```

### Fields

#### server

==Required==

The server address.

#### server_port

==Required==

The server port.

#### password

==Required==

QUICX user password

#### heartbeat

Interval for sending heartbeat packets for keeping the connection alive

`10s` is used by default.

#### bbr_profile

BBR congestion control algorithm profile, one of `conservative` `standard` `aggressive`.

`standard` is used by default.

#### network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

The ALPN must be `h3`.

### 0-RTT

QUICX attempts a 0-RTT connection handshake whenever a session ticket from a
previous connection is available, saving one round trip when the tunnel is
re-established. As the protocol is fully multiplexed this is not impacting much
on the performance.

When the server rejects the attempt (for example after a restart, a session
ticket key change, or an anti-replay rejection), the client resends the
authentication and the first request after the handshake completes instead of
closing the connection.

!!! warning ""
    0-RTT data is vulnerable to replay attacks, which matters for non-idempotent
    requests. The server only accepts it because the transport is indistinguishable
    from a standard HTTP/3 server.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
