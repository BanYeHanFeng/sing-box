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
    "scheme": "auto",
    "max_overhead_percent": 10,
    "max_group_size": 64,
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
expected (see [QUICX FEC](../quicx-fec.md#8-logging)).

While FEC is enabled, a statistics line is written every 10 seconds (windows without FEC
activity are skipped). It is written at debug level, and promoted to info level - at
most once a minute per connection - when packets were actually repaired:

```
QUICX FEC: tx loss 3.4% (peer reported), group 13 rows 2, overhead 15.4% configured / 6.0% measured,
  protected 1200 pkts (195.3 KB), parity 96 pkts (14.2 KB), skipped 2 groups, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 91 pkts, protected 1400 pkts
```

The `tx` column is the direction this endpoint **sends** on (the loss rate is measured by
the peer and reported back); the `rx` column is the direction it **receives** on. They are
measured by different endpoints, so don't read them as one number. `skipped` counts the
groups that were deliberately left unprotected to stay within the cap.

With the sliding window scheme, `group … rows …` becomes `window N pkts`, the configured
value becomes `rate x%`, and `skipped` counts rows:

```
QUICX FEC: tx loss 3.4% (peer reported), window 64 pkts, rate 5.1% / 4.8% measured,
  protected 1200 pkts (1.4 MB), parity 61 pkts (76.3 KB), skipped 2 rows, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 58 pkts, protected 1400 pkts
```

#### fec.enabled

Whether to enable FEC. Defaults to `true`; set it to `false` to disable FEC on this
endpoint.

#### fec.scheme

Which FEC scheme to use.

- `auto` (the default): this endpoint supports both the **sliding window scheme** and the
  **block scheme**, and the server picks the best one both endpoints implement (new
  connections therefore use the sliding window scheme);
- `window`: only the sliding window scheme;
- `block`: only the block scheme.

When the two endpoints have no scheme in common, FEC stays off on both sides and the
connection works as usual. See
[QUICX FEC](../quicx-fec.md#10-the-sliding-window-scheme) for the negotiation.

#### fec.max_overhead_percent

Upper bound of the parity traffic, as a percentage of the protected traffic.

`10` is used by default. The bound applies to the parity **bytes actually sent**: when a
group is too small, the packet sizes are too skewed, or (with the sliding window scheme)
the byte budget doesn't cover a row, FEC skips that parity (see `skipped` in the statistics
line) instead of exceeding the bound.

#### fec.max_group_size

Block scheme: the maximum number of packets protected by one FEC group, `32` by default.
Larger groups reduce the relative overhead, but increase the time until a lost packet can
be repaired.

Sliding window scheme: the window size, `64` by default. Every packet stays in the window
for that many packets and is covered by about `window * redundancy` rows, so a larger
window recovers longer bursts - at the cost of memory (about `window * MTU` per direction)
and parity computation.

With the block scheme the value has to be large enough to fit `max_parity_rows` parity rows
within the cap, otherwise the extra rows are never used (with the default `32` / `10%`, two
rows cost about 6.8% and do get used).

#### fec.max_parity_rows

Block scheme: the maximum number of parity rows per group.

`2` (the default) repairs two losses per group (Reed-Solomon parity over GF(2^8),
RAID 6 style); `1` repairs only a single loss per group (XOR parity, RAID 5 style).

Losses on mobile paths come in bursts: with a single row, a group with two or more
missing packets loses all of them (in a 15.7 hour production log `unrecoverable` was 2 to
2.5 times `repaired`), which is why two rows are the default.

Sliding window scheme: the number of repair rows an idle sender emits for the tail of its
window (`2` by default).

#### tls

==Required==

TLS configuration, see [TLS](/configuration/shared/tls/#inbound).

The ALPN must be `h3`.

### QUIC Fields

See [QUIC Fields](/configuration/shared/quic/) for details.

Standard HTTP/3 requests that are not QUICX proxy traffic are terminated with a
graceful HTTP/3 shutdown (GOAWAY, drain, then `H3_NO_ERROR`), matching the
behavior of a normal HTTP/3 server.
