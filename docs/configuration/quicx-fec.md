# QUICX FEC: surviving packet loss with erasure codes

This page answers one question: can QUICX use a "hard disk style" checksum (RAID-like
erasure coding, i.e. forward error correction) to survive packet loss, instead of
sending blindly at a fixed rate the way Hysteria2's `brutal` congestion control does -
and without wasting bandwidth?

The answer is **yes**. It is implemented and verified in CI over a lossy link. The core
idea is adaptive redundancy: no parity packet at all is sent on a path that doesn't lose
packets, and on a lossy path the amount of redundancy follows the measured loss rate,
capped by a configurable bound.

There are two schemes, negotiated during the QUICX handshake (section 10):

- the **sliding window scheme** (the default for new connections): repair rows are
  generated continuously for the most recent W packets that carry application data, so a
  lost packet is covered by many rows and bursts are reconstructed row by row;
- the **block scheme** (sections 2 to 9 describe it): a closed group of packets is
  protected by a fixed number of parity rows. Peers that don't know the window frame fall
  back to it, and it is never used when both endpoints support the window scheme.

## 1. How Hysteria2 survives loss, and what it costs

Hysteria2's `brutal` congestion control sends at a fixed, user configured rate and simply
ignores packet loss:

- the send rate never reacts to ACKs or losses;
- lost packets are repaired by QUIC retransmission, and the sender never backs off;
- the cost: set the rate too high and the excess traffic is dropped by the bottleneck
  (pure waste), set it too low and the link stays idle. A constant high rate is also very
  visible to QoS and to network operators.

QUICX (without FEC) uses a BBR-like congestion controller: losses are seen, the send rate
is reduced, and recovery costs roughly a round trip. FEC repairs the loss instead, without
adding pointless traffic.

## 2. From RAID to packet level erasure codes

- RAID 5: k data blocks plus one XOR parity block - any single missing block can be
  recomputed;
- RAID 6 / Reed-Solomon: k data blocks plus m parity blocks - any m missing blocks can be
  recomputed (a linear system over GF(2^8)).

Mapped to QUIC, a "block" is a **fully encrypted QUIC packet** (header included):

- the sender groups k consecutive 1-RTT packets, computes m parity packets (`m = 1` is
  plain XOR, RAID 5 style; `m >= 2` is a Vandermonde/Reed-Solomon parity over GF(2^8),
  RAID 6 style) and sends them as regular QUIC packets;
- the receiver caches the packets it recently received. When a parity packet arrives and a
  member of its group is missing, it reconstructs the **exact wire bytes** of the missing
  packet and feeds them back into the normal decryption and packet handling path;
- the repaired packet is acknowledged like a packet that arrived on the wire, so the
  sender's loss detection never sees it: no retransmission, no congestion window
  reduction.

Packets in a group have different sizes, so members are zero-extended to the longest
member before the parity is computed; the length of every member travels in the repair
frame (1-2 bytes each), which is what allows the receiver to truncate a recovered packet
back to its original size.

## 3. Why this works in QUIC

| Problem | Solution |
| --- | --- |
| Packet numbers / header protection | The parity is computed over the **sealed packet bytes**, so a recovered packet is directly decryptable |
| Variable packet sizes | Zero extension plus a length list in the repair frame |
| The parity packet must fit the MTU | Protected packets are limited to `MTU - header reserve` (about 63 bytes with `max_group_size=16`), so the parity packet fits |
| Recovery must happen before retransmission | The receiver caches roughly `2 x max_group_size` packets; the parity packet is sent right after a group closes, usually well before loss detection (>= 1 RTT) |
| Late / duplicate packets | A recovered packet that later arrives for real is dropped by QUIC's regular duplicate detection |
| Congestion accounting | Parity packets use the normal send path: they count toward bytes in flight and the congestion window, and are only sent when the send loop is allowed to send (they never bypass congestion control) |
| Key updates | The cache is dropped when the key phase changes |
| Compatibility / camouflage | No QUIC transport parameter is used (a private transport parameter would show up in the ClientHello and break the Chrome / HTTP/3 fingerprint). FEC is negotiated in QUICX's own control plane, and a peer that doesn't know the FEC frames **ignores** them instead of failing the connection |

## 4. Adaptive redundancy: no blind bandwidth

This is the main difference to `brutal`:

1. the **receiver measures** the packet loss rate of the path from gaps in the 1-RTT
   packet number sequence (a packet only counts as lost once it is 8 packets behind,
   so reordering is not mistaken for loss);
2. it reports the counters **every 20ms** (cumulative counters, so a lost report frame
   costs accuracy for nothing);
3. the **sender** smooths the reports with an EWMA (alpha = 0.3) and asks for
   `1.5 x loss rate` redundancy. It uses two Reed-Solomon parity rows when the loss rate
   is at least 12% (or the requirement at least 25%) and the group size limit can pay for
   them, one row otherwise. It derives the group size `k` from the requirement, counting
   the `FEC_REPAIR` frame header (estimated from an EWMA of the packet size) - without
   that, a group that is exactly at the cap would be rejected by the budget for its
   header alone and FEC would stop working precisely where it is needed most;
4. below 0.2% loss FEC is **completely off**: `k = 0`, not a single parity packet
   (verified in CI: `ParityPacketsSent = 0` on a clean path);
5. a partial group is protected after 2ms of idle time, so the last packets of a burst
   aren't left unprotected;
6. all of the above only decides what FEC *wants* to send. Every group also has to **pay
   for its own parity** (section 5); when it can't, it stays unprotected (counted as
   `skipped`) instead of overspending.

## 5. Overhead, and how the cap is actually enforced

**The cap bounds bytes, and it is applied per group.** A group is only protected when
`parity bytes <= group bytes * max_overhead_percent`; otherwise it is left unprotected and
counted in `skipped`. Because every group stays within the cap, so does **any** aggregation
of them - a window, a connection - which is what makes the `measured` value in the
statistics line directly checkable against the cap.

The check is per group rather than against a running budget on purpose. A running budget
also bounds the long run ratio, but it lets credit earned earlier pay for groups that are
structurally not worth protecting: packets so small that the `FEC_REPAIR` header dominates,
and groups too small to be worth a parity packet. A production run showed "2 packets (54 B)
protected by 1 parity packet (41 B)", and in its worst window 94 of 375 protected packets
could not be repaired at all. Those groups are unlikely to be repaired whatever FEC does
for them, so the bandwidth is better left unspent.

Within that budget, redundancy is about `m / k` (m parity rows over k data packets). With
the defaults `k <= 32`, `m <= 2`, and 1200 byte packets:

| Measured loss p | Required redundancy 1.5p | Rows m | Group k | Nominal overhead m/k | Two losses in a group |
| --- | --- | --- | --- | --- | --- |
| < 0.2% | - | - | off | 0 | - |
| 1% | 1.5% | 1 | 32 | 3.1% | not repaired |
| 2% | 3% | 1 | 32 | 3.1% | not repaired |
| 5% | 7.5% | 1 | 13 | 7.7% | not repaired |
| 10% | 15% (capped at 10%) | 1 | 11 | 9.1% | not repaired |
| 12% | 18% (capped at 10%) | 2 | 22 | 9.1% | **repaired** |
| >= 20% | 30% (capped at 10%) | 2 | 22 | 9.1% | **repaired** |

Notes:

- redundancy comes in steps of `1/k`, so at very low loss rates the actual overhead is a
  few times the loss rate (3.1% at 1%). Larger `k` lowers the floor, at the cost of a
  longer repair delay and a higher chance of a burst landing inside one group;
- **two parity rows have to fit**: `2/max_group_size` must be within
  `max_overhead_percent`, otherwise that branch is never taken. With the default `32` /
  `10%` two rows cost about 6.8% including the frame header and are used; with
  `max_group_size: 16` two rows need `max_overhead_percent >= 13`;
- **a single row can't repair bursts**: two or more losses in one group leave the whole
  group unrepaired (the decoder counts *every* missing packet of such a group as
  `unrecoverable`). Losses on mobile paths come in bursts, which is why two rows are now
  the default;
- **small packets and small groups are left unprotected on purpose**: a parity packet
  carries a frame header (up to 112 bytes with `max_group_size=32`) plus the short header
  and AEAD tag every QUIC packet spends (about 30-50 bytes), so with tiny packets or tiny
  groups the parity would exceed 10% of that group's bytes and FEC skips it (`skipped`).
  Refusing to repair one group beats overspending. The flip side: **packets of around
  27 bytes cannot be protected at all under a 10% cap** - that traffic relies on QUIC
  retransmission, or on a larger `max_overhead_percent`;
- FEC also reduces the maximum protected packet size by one frame header reserve plus one
  packet overhead (about 112 + 48 bytes with `max_group_size=32`). That cost exists only
  while FEC is active, i.e. only on lossy paths.

## 6. Comparison with HY2 `brutal`

| | HY2 `brutal` | QUICX FEC |
| --- | --- | --- |
| Mechanism | Fixed high send rate + retransmission | Proactive erasure coding + retransmission as a fallback |
| Bandwidth cost | Bound to the configured rate, permanently | About the measured loss rate, bounded by a configured cap |
| Idle / clean path | Still sends at the configured rate | **Zero redundancy** |
| Congestion control | Bypassed | Fully respected, parity counts toward cwnd |
| Recovery latency | About one round trip | As soon as the parity packet arrives |
| Bursty loss | Retransmission | One loss per group (two with `max_parity_rows: 2`) |
| Traffic signature | Constant high rate, easy to spot | Same shape as regular QUIC traffic, plus a few small packets |

## 7. Measurements on a real lossy path

Next to the CI tests, end-to-end measurements were run with a binary built by the release
pipeline, over a `veth` link with `tc netem loss 12%` (per packet, in both directions,
GSO enabled): a 3MB HTTP download through the QUICX proxy and 150 UDP round trips.

| Configuration | TCP 3MB median | UDP delivery (150 round trips) |
| --- | --- | --- |
| FEC off | 233 ms | 109/150 = 72.7% |
| FEC on (default 10% cap) | 241 ms | **116/150 = 77.3%** |
| FEC on (25% cap) | 299 ms | 116/150 = 77.3% |

- **UDP / DATAGRAM relay**: FEC directly improves delivery (+4.6 percentage points here).
  DATAGRAM frames are never retransmitted, so a lost one is lost forever; erasure coding is
  the only way to get it back on the receiver side.
- **TCP streams**: QUIC streams retransmit and BBR tolerates 12% loss; FEC doesn't help
  throughput there. At the default 10% cap it is roughly free (241ms vs 233ms), at a 25% cap
  it is clearly slower (299ms) - the extra parity is pure waste. For interactive,
  request/response traffic the win is one round trip of recovery latency, not throughput.
- Recommended deployment: prefer FEC for UDP. Either use two outbounds with
  `"network": "udp"` (FEC on) and `"network": "tcp"` (FEC off), or enable it globally with
  the default 10% cap.

## 8. Logging

Both sides log a debug line once FEC is negotiated, including the peer address:

```
QUICX FEC enabled (server, 203.0.113.9:41234, max overhead 10%, max group 32, parity rows 2)
QUICX FEC enabled (client, 198.51.100.7:30010, max overhead 10%, max group 32, parity rows 2)
```

**One line per QUIC connection, not per client or per process.** FEC is negotiated per
connection (the client announces support in its authentication request and the server
switches that connection to FEC after confirming it), so every new connection negotiates
it again: a client that redials on heartbeat, idle timeout or network change adds one
such pair to the log. A QUICX outbound keeps a single connection and redials when it
drops, so dozens of lines in one log file are expected and do not mean that FEC was
enabled repeatedly on one connection.

- Don't use this line to tell whether FEC is working; watch the statistics below
  instead - a growing `repaired` is the actual evidence;
- If the server can't open the confirmation stream, it logs a `notify FEC accept` error:
  the client doesn't enable FEC and the server stops short of logging `enabled`. Only the
  client side `enabled (client, ...)` line and the statistics prove that both ends
  agreed;
- The line is written at **debug** level only. To keep it out of a frequently redialing
  client's log, use the default `"log": {"level": "info"}` - the info level statistics
  line for repaired windows is not lost.

While FEC is enabled, a statistics line is written every 10 seconds (windows without
any FEC activity are skipped):

```
QUICX FEC: tx loss 3.4% (peer reported), group 13 rows 2, overhead 7.7% configured / 7.2% measured,
  protected 1200 pkts (1.4 MB), parity 96 pkts (118.2 KB), skipped 2 groups, dropped 0 frames;
  rx repaired 128, unrecoverable 9, parity 91 pkts, protected 1400 pkts
```

- The line is written at **debug** level, except when packets were actually repaired
  (`repaired`/`unrecoverable` > 0): those windows are logged at **info** level, at most
  once a minute per connection, so a lossy server doesn't flood its log. In other words,
  the default `"log": {"level": "info"}` is enough to see whether FEC is doing
  something; use `"log": {"level": "debug"}` to also see the idle (zero redundancy)
  windows.
- **`tx` and `rx` are two directions measured by different endpoints - don't read them as
  one number**:
  - `tx loss` is the loss rate of the direction this endpoint **sends** on, measured by the
    **peer** from packet number gaps (including packets FEC repaired, i.e. the real path
    quality). The `protected`/`parity`/`overhead`/`skipped` fields next to it describe this
    endpoint's sending side;
  - `rx repaired`/`rx unrecoverable` are what this endpoint's decoder saw on the direction
    it **receives** on (`unrecoverable` counts only packets parity couldn't repair, which
    fall back to QUIC retransmission); `rx parity`/`rx protected` are the received parity
    packet and protected packet counts.
- `overhead ... configured / ... measured`: the first value is the `m/k` **configuration**,
  the second is the **measured** `parity bytes / protected bytes`. The cap applies to the
  measured value; printing only the configured one hides problems such as a partial group
  being protected at 50% overhead.
- `skipped N groups`: groups this window that were deliberately left unprotected to stay
  within the cap. A non-zero value means the cap is doing its job; a persistently large one
  means the packets or groups are too small - raise `max_group_size` or the cap.
- `dropped N frames`: parity frames discarded this window because the send queue stayed
  busy for too long. It should be zero; a growing value means FEC is not actually
  protecting anything on that side (a saturated upload or download), so check whether the
  link is simply maxed out.

## 9. Configuration

FEC is **enabled by default**: omitting the `fec` section keeps it enabled with the
defaults, and `"fec": {"enabled": false}` disables it on that endpoint.

```json
{
  "type": "quicx",
  "fec": {
    "enabled": true,
    "scheme": "auto",
    "max_overhead_percent": 10,
    "max_group_size": 32,
    "max_parity_rows": 2
  }
}
```

FEC has to be enabled on both sides; the client announces support in its authentication
request and only enables FEC once the server confirmed it, so enabling it on one side
never wastes bandwidth. See the [outbound](outbound/quicx.md#fec) and
[inbound](inbound/quicx.md#fec) pages for the individual fields.

## 10. The sliding window scheme

The block scheme has two structural weaknesses that parameter tuning cannot remove, and
both showed up in production logs:

- **All or nothing per group.** A group of k packets protected by m parity rows is either
  fully repaired or not repaired at all: a group that loses more than m packets gives up
  every one of its missing packets to retransmission. Mobile paths lose packets in bursts,
  so most groups lose one or two packets while a few lose more - and the few dominate the
  `unrecoverable` count.
- **The tail and low rate flows cannot pay for parity.** A group is only protected when
  its own parity bytes fit into the overhead cap, so the last packets before an idle
  period, and the flows whose packets are small or sparse, are skipped - on a path that is
  losing packets exactly then.

The sliding window scheme replaces the group with a **window**:

- the sender keeps the last `max_group_size` packets (64 by default) that carry
  application data, and emits one repair row per ~`1/rate` packets, where
  `rate = min(1.5 * measured loss, what the cap allows)`. Below 0.2% loss `rate` is 0 and
  not a single parity packet is sent. An idle sender (2ms) emits one or two rows for the
  tail of its window;
- **every row protects the whole current window**, so a lost packet is covered by about
  `W * rate` rows instead of exactly m: a burst is reconstructed row by row as the rows
  arrive, rather than depending on one group;
- a row carries a **bitmap** of member packet numbers (about one bit per packet number,
  since the packet numbers spent on parity packets simply stay clear) plus a two byte
  **length parity** - the GF(2^8) combination of the members' wire lengths, so a recovered
  packet's length is solved together with the packet instead of being listed per member;
- the coefficients form a **Cauchy matrix** over GF(2^8): `1 / (x_row + y_position)`. Every
  square submatrix of a Cauchy matrix is invertible, so any n rows reconstruct any n
  members of the window for **any** loss pattern;
- the decoder turns each row into an equation over the packets it is still missing, does
  incremental Gaussian elimination, and reconstructs a packet as soon as an equation
  reduces to a single unknown. A reconstructed packet is then treated like a packet that
  arrived on the wire: it is acknowledged normally, and it is used as a known symbol by
  the rows that follow. A packet that arrives late is substituted into the pending
  equations as well, which can make a lost packet recoverable immediately;
- packets that only acknowledge packets, or that only update flow control state, are not
  protected: they carry information the peer already has, and their small size would still
  make every parity symbol as long as the data packets next to them.

**The overhead bound** is a byte credit rather than a per group decision: every protected
packet adds `cap * packet bytes` to the credit, and a row is only sent when the credit
covers its actual size. Because only earned bytes can be spent, the measured parity/protected
ratio stays within `max_overhead_percent` over the whole connection, and the `measured`
value in the statistics line can be checked against the cap directly.

**Memory.** The sender keeps a copy of every packet in its window, and the receiver caches
two windows plus slack, so a connection spends roughly two windows of MTU sized packets per
direction (about 300 KB at the default window of 64, about 600 KB at 128) - somewhat more
than the block scheme, which keeps one group plus a smaller cache.

**Capacity.** A packet stays in the window for W packets, during which about `W * rate`
rows cover it, so the recoverable burst length is about `W * rate` (with `rate` already
capped). With the default `W = 64`, 1200 byte packets and a 10% cap:

| measured loss p | target rate | covering rows | expected losses in the window | recoverable burst |
| --- | --- | --- | --- | --- |
| < 0.2% | 0 (idle) | 0 | - | - |
| 1% | 1.5% | 0.96 | 0.64 | ~1 |
| 2% | 3% | 1.9 | 1.3 | ~2 |
| 5% | 7.5% | 4.8 | 3.2 | ~4 |
| 10% | 9.7% (capped) | 6.2 | 6.4 | ~6 |
| 20% | 9.7% (capped) | 6.2 | 12.8 | ~6, the rest falls back to retransmission |

The information theoretic limit still applies: a 10% cap cannot repair 20% random loss. What
the window scheme changes is that a burst of up to about `W * rate` consecutive packets is
recoverable (the block scheme repairs at most m per group of k and gives up the whole group
when a burst is longer).

**Negotiation.** The client's capability byte is a bit mask (`0x01` block, `0x02` window),
the server picks the best scheme both endpoints support and echoes it after
`CommandFECAccept`. A server that only knows the block scheme ignores the window bit and
replies without a scheme byte, and the client falls back to the block scheme; when the two
endpoints have no scheme in common, FEC stays off on both sides. The three repositories can
therefore be upgraded in any order.

**Configuration.** `fec.scheme` is `auto` (default), `window` or `block`.
`max_group_size` is the number of packets per group (block) or the window size (window,
default 64); `max_parity_rows` is the number of parity rows per group (block) or the number
of rows an idle sender emits for the tail of its window (window).

**Verified in CI** (GitHub Actions, all green): a Cauchy MDS assertion over arbitrary row
and member combinations, single loss, four packet bursts, unequal packet sizes, a late
packet completing an underdetermined equation, the idle tail, acknowledgement-only packets
being ignored, the overhead cap, and repair frame round-trip/truncation/invalid input as
unit tests; plus end-to-end tests over real UDP with loss injection and `-race`: no parity
on a clean path, recovery at about 12% loss, recovery of bursts of three consecutive
packets, and measured overhead within the cap on both endpoints. The block scheme's own
tests are kept and still pass, so the compatibility path is intact.

**Not verified yet**: the window scheme's numbers on a real cross-border mobile path
(`repaired` / `unrecoverable` / `skipped` / throughput over hours), and the trade-off
between `window` sizes of 32 and 128. Run `"scheme": "window"` and `"scheme": "block"` side
by side for a while before changing the default window size.
