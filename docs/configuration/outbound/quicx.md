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
  "fec": {
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

This is the password-only credential used to authenticate to the server (the
user UUID has been removed).

#### heartbeat

Interval for sending heartbeat packets for keeping the connection alive

`10s` is used by default.

#### bbr_profile

BBR congestion control algorithm profile, one of `conservative` `standard` `aggressive`.

`standard` is used by default.

#### fec

Packet level forward error correction (FEC) configuration, used to repair QUIC
packets lost on the path. See [QUICX FEC](/configuration/quicx-fec/).

Leave it unset to disable FEC. FEC only takes effect if the server enables it as well:
the client announces support in its authentication request, and only enables FEC once
the server confirmed it. Enabling it on one side therefore never wastes bandwidth.

#### fec.max_overhead_percent

Upper bound of the parity traffic, as a percentage of the protected traffic.

`10` is used by default. FEC never exceeds this bound, no matter how lossy the path is.

#### fec.max_group_size

Maximum number of packets protected by one FEC group.

`16` is used by default. Larger groups reduce the relative overhead, but increase the
time until a lost packet can be repaired.

#### fec.max_parity_rows

Maximum number of parity rows per group.

`1` (the default) repairs a single loss per group (XOR parity, RAID 5 style), `2`
repairs two losses per group (Reed-Solomon parity over GF(2^8), RAID 6 style).

#### network

Enabled network

One of `tcp` `udp`.

Both is enabled by default.

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#outbound).

The ALPN must be `h3`.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

### Dial Fields

See [Dial Fields](/configuration/shared/dial/) for details.
