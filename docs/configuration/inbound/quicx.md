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
  "fec": {
    "enabled": true,
    "max_overhead_percent": 20,
    "max_group_size": 128,
    "max_parity_rows": 2
  },
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

`standard` is used by default.

#### fec

Packet level forward error correction (FEC) configuration, used to repair QUIC
packets lost on the path. See [QUICX FEC](../quicx-fec.md) for the mechanism, the
overhead math and measurements.

FEC is **enabled by default**: omitting this section keeps it enabled with the defaults
below. It only takes effect if both endpoints enable it: the client announces support in
its authentication request, and FEC is only turned on once the server confirmed it. FEC is
negotiated **per connection**, so both sides write one `QUICX FEC enabled` debug line
(including the peer address) for every connection - one more pair per client redial is
expected (see [QUICX FEC](../quicx-fec.md#5-logging)).

While FEC is enabled, a statistics line is written every 10 seconds (windows without FEC
activity are skipped). It is written at debug level, and promoted to info level - at
most once a minute per connection - when packets were actually repaired:

```
QUICX FEC: tx loss 3.4% (peer reported), window 128 pkts, rate 5.1% / 4.8% measured,
  protected 1200 pkts (1.4 MB), parity 61 pkts (76.3 KB), skipped 2 rows, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 58 pkts, protected 1400 pkts
```

The `tx` column is the direction this endpoint **sends** on (the loss rate is measured by
the peer and reported back); the `rx` column is the direction it **receives** on. They are
measured by different endpoints, so don't read them as one number. `skipped` counts the
repair rows that were deliberately left unsent to stay within the cap.

#### fec.enabled

Whether to enable FEC. Defaults to `true`; set it to `false` to disable FEC on this
endpoint.

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

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

The ALPN must be `h3`.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

Standard HTTP/3 requests that are not QUICX proxy traffic are terminated with a
graceful HTTP/3 shutdown (GOAWAY, drain, then `H3_NO_ERROR`), matching the
behavior of a normal HTTP/3 server.
