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
  "qlog_directory": "",
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

#### qlog_directory

Write a [qlog](https://datatracker.ietf.org/doc/draft-ietf-quic-qlog-main-schema/)
trace for every QUIC connection into this directory, named
`<connection id>_client.sqlog`.

Disabled by default. The directory is created at startup, and a path that
cannot be written makes sing-box fail to start instead of tracing nothing.

!!! note ""
    Every connection gets its own file and nothing prunes the directory, so it
    only grows. Enable this while debugging a specific problem, not for
    permanent operation.

The traces can be opened with [qvis](https://qvis.quictools.info/).

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
re-established. As the protocol is fully multiplexed only the first request
after the re-establishment benefits from it, and that request is what the
client's 0-RTT data carries: the authentication request, the CONNECT request
(the destination) and the first payload of the proxied connection are all sent
in the first flight.

When the server rejects the attempt (for example after a restart, a session
ticket key change, or an anti-replay rejection), the client resends the
authentication and the first request after the handshake completes instead of
closing the connection.

The client attaches a random nonce to the authentication request of every
connection, and the server remembers the nonces of authenticated sessions and
rejects a nonce another session already used (see 0-RTT and replay in the
inbound documentation). A replayed 0-RTT flight therefore fails authentication
and never establishes a connection to the destination.

!!! warning ""
    0-RTT data is vulnerable to replay attacks: whoever captured a client's 0-RTT
    flight can send it to the server again, and the transport cannot tell the copy
    from the original. The attacker does not get a usable tunnel out of it (he has
    no handshake keys, cannot decrypt responses and cannot construct 1-RTT data),
    but without the nonce check above the server would authenticate again, connect
    to the destination again and deliver the first payload a second time, which
    matters for non-idempotent requests (a plaintext HTTP POST, for example).

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
