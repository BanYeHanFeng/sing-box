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
    "max_overhead_percent": 30,
    "max_group_size": 128,
    "max_parity_rows": 2,
    "baseline_redundancy_percent": 5
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
most once a minute per connection - when the window is notable (packets were repaired or
given up on, a duplicate repair row arrived, or the sender went idle with protected
packets still missing):

```
QUICX FEC: tx loss 3.4% (peer reported), window 128 pkts, rate 5.1% / 4.8% measured,
  protected 1200 pkts (1.4 MB), parity 96 pkts (112.5 KB), skipped 2 rows (2 budget, 0 too large),
  dropped 0 frames, rtt 24.6ms (+3.8ms vs min);
  rx repaired 128, unrecoverable 9, parity 91 pkts, protected 1400 pkts
```

The `tx` column is the direction this endpoint **sends** on (the loss rate is measured by
the peer and reported back); the `rx` column is the direction it **receives** on. They are
measured by different endpoints, so don't read them as one number. `skipped` counts the
repair rows that were deliberately left unsent (the cap's byte credit didn't cover them,
or the row could not be built).

#### fec.enabled

Whether to enable FEC. Defaults to `true`; set it to `false` to disable FEC on this
endpoint (the server then won't enable it either).

#### fec.max_overhead_percent

Upper bound of the parity traffic, as a percentage of the protected traffic.

`30` is used by default. The bound applies to the parity **bytes actually sent**: when the
byte budget doesn't cover a repair row, FEC skips that row (see `skipped` in the statistics
line) instead of exceeding the bound.

#### fec.max_group_size

The window size, `128` by default (the largest window the wire format carries; a larger
value is clamped to `128`). Every packet stays in the window for that many packets
and is covered by about `window * redundancy` rows, so a larger window recovers longer
bursts - at the cost of memory (roughly two windows of MTU sized packets per direction)
and parity computation.

#### fec.max_parity_rows

The number of repair rows an idle sender emits for the tail of its window (`2` by default;
a larger value is clamped to `2`).

How many rows cover a packet while it stays in the window depends on the redundancy; the
packets sent last are covered by the fewest rows, and the idle tail rows are there for
them. `1` reduces the parity traffic of an idle connection, `2` repairs one more loss in
the tail.

#### fec.baseline_redundancy_percent

The baseline redundancy rate in percent (default `5`). It keeps a small share of parity
flowing even while the path looks clean, for links that occasionally burst to around 8%
loss and then drop back to clean: the first loss does not have to wait for the peer's
report (about 0.5*RTT plus sampling delay). Set it explicitly to `0` to return to a purely
reactive path. It is still bounded by `max_overhead_percent`; `5` is the recommended knee
for ~8% bursts, `3` is cheaper on a clean path, and `8` protects more of the initial burst.

#### fec.recovered_packet_feedback

Report packets this endpoint reconstructed with FEC back to the sender (`false` by default).

When enabled, the receiver sends a `FEC_RECOVERED` frame after repairing packets, and the
sender feeds the corresponding loss to its congestion controller without retransmitting
them (they were already processed and acknowledged as received). This keeps FEC from
hiding the congestion signal. Both ends must understand the frame, so it is off by
default; enable it on both ends only if FEC connections should take part in congestion
control feedback.

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
