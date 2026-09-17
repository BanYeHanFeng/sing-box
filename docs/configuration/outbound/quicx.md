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
    "enabled": true,
    "max_overhead_percent": 20,
    "max_group_size": 128,
    "max_parity_rows": 2
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
packets lost on the path. See [QUICX FEC](../quicx-fec.md) for the mechanism, the
overhead math and measurements.

FEC is **enabled by default**: omitting this section keeps it enabled with the defaults
below. It only takes effect if the server enables it as well: the client announces
support in its authentication request, and only enables FEC once the server confirmed
it. Enabling it on one side therefore never wastes bandwidth. FEC is negotiated **per
connection**, so both sides write one `QUICX FEC enabled` debug line (including the peer
address) for every connection - one more pair per redial is expected
(see [QUICX FEC](../quicx-fec.md#5-logging)).

While FEC is enabled, a statistics line is written every 10 seconds (windows without FEC
activity are skipped). It is written at debug level, and promoted to info level - at
most once a minute per connection - when packets were actually repaired:

```
QUICX FEC: tx loss 3.4% (peer reported), window 128 pkts, rate 5.0% / 4.8% measured,
  protected 1200 pkts (1.4 MB), parity 96 pkts (112.5 KB), skipped 2 rows, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 91 pkts, protected 1400 pkts
```

The `tx` column is the direction this endpoint **sends** on (the loss rate is measured by
the peer and reported back); the `rx` column is the direction it **receives** on. They are
measured by different endpoints, so don't read them as one number. `skipped` counts the
repair rows that were deliberately left unsent to stay within the cap.

#### fec.enabled

Whether to enable FEC. Defaults to `true`; set it to `false` to disable FEC on this
endpoint (the server then won't enable it either).

#### fec.max_overhead_percent

Upper bound of the parity traffic, as a percentage of the protected traffic.

`20` is used by default. The bound applies to the parity **bytes actually sent**: when the
byte budget doesn't cover a repair row, FEC skips that row (see `skipped` in the statistics
line) instead of exceeding the bound.

#### fec.max_group_size

The window size, `64` by default. Every packet stays in the window for that many packets
and is covered by about `window * redundancy` rows, so a larger window recovers longer
bursts - at the cost of memory (roughly two windows of MTU sized packets per direction)
and parity computation.

#### fec.max_parity_rows

The number of repair rows an idle sender emits for the tail of its window (`2` by default).

How many rows cover a packet while it stays in the window depends on the redundancy; the
packets sent last are covered by the fewest rows, and the idle tail rows are there for
them. `1` reduces the parity traffic of an idle connection, `2` repairs one more loss in
the tail.

#### fec.baseline_redundancy_percent

The baseline redundancy rate in percent (default `0`: a clean path sends no parity at all).

A purely reactive FEC only starts protecting after the peer reports the first loss. On a
high-RTT or low-rate path the first burst can already have left the window by then. Setting
this to `2`-`5` keeps that share of parity traffic flowing even while the path looks clean,
so the first loss has repair rows immediately; the price is that a clean path is no longer
idle. The value is still bounded by `max_overhead_percent`. Prefer small values, and only
enable it on high-RTT or low-rate paths.

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
