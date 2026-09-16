# QUICX FEC: surviving packet loss with erasure codes

This page answers one question: can QUICX use a "hard disk style" checksum (RAID-like
erasure coding, i.e. forward error correction) to survive packet loss, instead of
sending blindly at a fixed rate the way Hysteria2's `brutal` congestion control does -
and without wasting bandwidth?

The answer is **yes**. It is implemented and verified in CI over a lossy link. The core
idea is adaptive redundancy: no parity packet at all is sent on a path that doesn't lose
packets, and on a lossy path the amount of redundancy follows the measured loss rate,
capped by a configurable bound.

There is one scheme: the **sliding window scheme**. Repair rows are generated
continuously for the most recent W packets that carry application data, so a lost packet is
covered by many rows and bursts are reconstructed row by row. The **block scheme** this
fork used to implement (a closed group of packets protected by a fixed number of parity
rows) has been removed from quic-go, sing-quic and sing-box: its two structural weaknesses
cannot be fixed by tuning, while the window scheme covers all of its capabilities. This
page therefore describes the current implementation only; the design notes and measurements
of the removed scheme were deleted along with it (section 8).

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

## 2. The sliding window scheme

The scheme keeps a continuously rolling **window** of the most recent packets (the idea
behind the elastic window of RFC 9407 Tetrys and the IETF sliding window RLC drafts,
implemented here with deterministic coefficients and a length parity) instead of closing
packets into fixed groups.

### 2.1 Sender

- the window is the last `max_group_size` packets (128 by default) that carry application
  data. Packets that only acknowledge packets, or that only update flow control state, are
  not added: they carry information the peer already has, and since a parity symbol is as
  long as the longest member of the window, mixing in small packets would only make every
  row as long as the data packets next to them;
- one repair row is emitted per ~`1/rate` packets, where
  `rate = min(1.5 * measured loss, what the cap allows)`. Below 0.2% loss `rate` is 0 and
  not a single parity packet is sent;
- an idle sender (2ms, `flush_delay`) emits one or two rows for the tail of its window, so
  the last packets of a burst aren't left with very few covering rows;
- **every row protects the whole current window**, so a packet is covered by all the rows
  that follow it: a loss is not "one group's problem" but has a chance to be repaired by
  the next `W / step` rows.

### 2.2 Repair row format and coefficients

```
FEC_WINDOW_REPAIR (0x34):
  row | first packet number | span | member bitmap | member count | parity length | length parity (2B) | parity data
```

- a row carries a **bitmap** of member packet numbers instead of a packet number list:
  packet numbers inside a window are nearly consecutive (the numbers spent on parity
  packets simply stay clear), so one bit per packet number - about 9-10 bytes at
  `window=64`;
- **lengths are not listed per member**: the members' wire lengths are combined with the
  same coefficients over GF(2^8) and only two bytes are sent. The length is 16 bit and its
  two bytes are solved by the same equations as the packet, so recovering a packet recovers
  its length at the same time;
- the coefficients form a **Cauchy matrix** over GF(2^8): `c(row, position) = 1 / (x_row +
  y_position)`, with row basis `x = α^0..α^63` and position basis `y = α^64..α^191`
  (disjoint, so no denominator is zero). Every square submatrix of a Cauchy matrix is
  invertible, so any n rows reconstruct any n members of the window for **any** loss
  pattern.

### 2.3 Receiver: incremental elimination and peeling

- when a repair row arrives, the contributions of the members that did arrive are
  subtracted from it, which leaves one equation over the packets that are still missing;
- the equations are reduced by incremental Gaussian elimination: rows with the same pivot
  are subtracted from each other, and a row that is down to a single unknown solves that
  packet directly;
- **a reconstructed packet is treated like a packet that arrived on the wire**: it goes
  back into the normal decryption and packet handling path (and is acknowledged normally),
  and it is substituted into the remaining equations as a known symbol - so a burst is
  peeled apart packet by packet as later rows arrive, instead of depending on one group;
- **late packets are substituted as well**: a packet that arrives late can turn an
  underdetermined equation into a solvable one immediately, without waiting for the next
  row;
- packets that are still missing after one second are counted as `unrecoverable`
  (`FailedPackets`) and fall back to QUIC retransmission.

### 2.4 How the overhead cap is enforced

The window scheme uses a **byte credit**: every protected packet adds `cap * packet bytes`
to the credit, and sending a row subtracts that row's actual bytes. A row the credit cannot
pay for is skipped (counted in `skipped`). Because only earned bytes can be spent,

> `parity bytes / protected bytes <= max_overhead_percent` holds **over the whole
> connection**, and the `measured` value in the statistics line can be checked against the
> cap directly.

The credit is capped (32 KB) so that credit earned on a long clean stretch is not dumped in
one burst; what limits spending in normal operation is the target redundancy rate `rate`
(about 1.5 x loss), and the credit only enforces "never more than the cap".

### 2.5 Capacity

A packet stays in the window for W packets, during which about `W * rate` rows cover it, so
the recoverable burst length is about `W * rate` (with `rate` already capped). With the
default `W = 128`, 1200 byte packets and a 10% cap:

| measured loss p | target rate | covering rows | expected losses in the window | recoverable burst |
| --- | --- | --- | --- | --- |
| < 0.2% | 0 (idle) | 0 | - | - |
| 1% | 1.5% | 1.9 | 1.3 | ~2 |
| 2% | 3% | 3.8 | 2.6 | ~4 |
| 5% | 7.5% | 9.6 | 6.4 | ~9 |
| 10% | 9.4% (capped) | 12.0 | 12.8 | ~12 |
| 20% | 9.4% (capped) | 12.0 | 25.6 | ~12, the rest falls back to retransmission |

The information theoretic limit still applies: a 10% cap cannot repair 20% random loss, and
a burst longer than about twelve consecutive packets is not repaired either. The extra rows
are still not wasted when a burst is too long: they repair the packets they can, and the
rest falls back to retransmission.

Two details decide whether that capacity is actually used on a bursty path. The redundancy
is derived from the **peak loss rate of the last second**, not from the smoothed estimate:
a burst is reported once and the reports after it are clean, and the smoothed estimate only
moves a fraction of the way to a sample, so it would both under-drive the redundancy and
decay while the packets of the burst are still inside the window. The window is also as
large as it gets: a shorter window cannot pay for a burst of this length even with the
credit it accumulated, because the rows a burst needs have to be spent before its packets
leave the window.

### 2.6 Negotiation

- the client appends a one byte capability flag to its authentication request: `0x02` is
  the sliding window scheme (the only scheme this version implements);
- the server appends the selected scheme after `CommandFECAccept` (also `0x02`), and the
  client only enables FEC when it reads that byte back and it matches the scheme it
  announced;
- an **old client** that only announces the block scheme flag (`0x01`) has no scheme in
  common with a new server, so **FEC stays off on both sides** and the connection works as
  usual;
- a new client talking to an **old server** that only knows the block scheme fails to
  negotiate in the same way, so FEC stays off (it is never sent window frames it cannot
  decode);
- an intermediate version that implemented both schemes and announced `0x03` still
  interoperates: it supports `0x02`, so both ends run the window scheme;
- the confirmation **must** carry the scheme byte: a client that cannot read it, or reads a
  different value, does not enable FEC (it never falls back to enabling one scheme by
  default).

## 3. Comparison with HY2 `brutal`

| | HY2 `brutal` | QUICX FEC |
| --- | --- | --- |
| Mechanism | Fixed high send rate + retransmission | Proactive erasure coding + retransmission as a fallback |
| Bandwidth cost | Bound to the configured rate, permanently | About the measured loss rate, bounded by a configured cap (zero on a clean path) |
| Idle / clean path | Still sends at the configured rate | **Zero redundancy** |
| Congestion control | Bypassed | Fully respected, parity counts toward cwnd |
| Recovery latency | About one round trip | As soon as a repair row arrives (peeled row by row) |
| Bursty loss | Retransmission | About `W * rate` consecutive losses, peeled row by row |
| Traffic signature | Constant high rate, easy to spot | Same shape as regular QUIC traffic, plus a few small packets |
| Fit | Lossy, long haul paths | The same, but when not burning bandwidth or attracting QoS matters |

## 4. Configuration

FEC is **enabled by default**: omitting the `fec` section keeps it enabled with the
defaults, and `"fec": {"enabled": false}` disables it on that endpoint. It only takes
effect if both endpoints enable it: the client announces support in its authentication
request and FEC is only turned on once the server confirmed it.

```json
{
  "type": "quicx",
  "fec": {
    "enabled": true,
    "max_overhead_percent": 10,
    "max_group_size": 128,
    "max_parity_rows": 2
  }
}
```

- `max_group_size`: the window size (128 by default);
- `max_parity_rows`: the number of repair rows an idle sender emits for the tail of its
  window (2 by default, at most 2);
- `max_overhead_percent`: the byte ratio cap for the whole connection (credit based):
  every protected packet adds `cap * packet bytes` to the credit (capped at 32 KB in
  total), and every row subtracts its actual bytes;
- `fec.scheme` has been removed: there is only one scheme to run, and a config that sets
  the field is rejected as an unknown field.

See the [outbound](outbound/quicx.md#fec) and [inbound](inbound/quicx.md#fec) pages for
the individual fields.

## 5. Logging

Both sides log a debug line once FEC is negotiated, including the peer address:

```
QUICX FEC enabled (server, 203.0.113.9:41234, sliding window scheme, max overhead 10%, window 128, tail rows 2)
QUICX FEC enabled (client, 198.51.100.7:30010, sliding window scheme, max overhead 10%, window 128, tail rows 2)
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
QUICX FEC: tx loss 3.4% (peer reported), window 128 pkts, rate 7.7% / 7.2% measured,
  protected 1200 pkts (1.4 MB), parity 96 pkts (118.2 KB), skipped 2 rows, dropped 0 frames;
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
    quality). The `protected`/`parity`/`rate`/`skipped` fields next to it describe this
    endpoint's sending side;
  - `rx repaired`/`rx unrecoverable` are what this endpoint's decoder saw on the direction
    it **receives** on (`unrecoverable` counts only packets parity couldn't repair, which
    fall back to QUIC retransmission); `rx parity`/`rx protected` are the received parity
    packet and protected packet counts.
- `rate ... / ... measured`: the first value is the sender's current target redundancy
  rate (about `1.5 * measured loss`, lowered by the cap), the second is the **measured**
  `parity bytes / protected bytes`. The cap applies to the measured value, and since it is
  accumulated over the whole connection, the `measured` value cannot exceed the cap.
- `skipped N rows`: repair rows this window that were deliberately not sent because the
  byte credit couldn't pay for them. A non-zero value means the cap is doing its job; a
  persistently large one means the packets or the flow are too small - consider raising
  `max_overhead_percent`.
- `dropped N frames`: parity frames discarded this window because the send queue stayed
  busy for too long. It should be zero; a growing value means FEC is not actually
  protecting anything on that side (a saturated upload or download), so check whether the
  link is simply maxed out, or lower the FEC window size.

## 6. Limitations

- only 1-RTT application data packets are protected; the handshake and 0-RTT are not (the
  handshake has retransmission of its own);
- only packets that carry application data are protected; packets that only acknowledge
  packets or update flow control state are not added to the window (protecting them has no
  value, and it would only make every row as long as the data packets next to them);
- parity packets are lost as well (at loss rate p, redundancy is effective about `1 - p` of
  the time), so FEC improves the delivery probability, it doesn't guarantee it;
- how many rows cover a packet is decided by the redundancy rate (about
  `window * rate` rows, minus the rows that are lost themselves); when the measured loss
  rate approaches the redundancy the cap allows, there are not enough equations and
  noticeably more packets fall back to QUIC retransmission. That is the deliberate
  trade-off of the cap: better to leave a packet unrepaired than to overspend;
- QUIC streams retransmit, so the value of FEC for TCP traffic is **saving a round trip of
  recovery latency and avoiding a congestion control misjudgement**, not replacing
  retransmission (for UDP/DATAGRAM relay FEC is the only way to get a lost packet back,
  since DATAGRAM frames are never retransmitted);
- with very small packets or a very sparse flow the credit never accumulates enough and the
  row header is a large share of the row, so FEC skips the row (`skipped`);
- to be evaluated: adapting the window size to the RTT, long runs on real mobile networks,
  and a more conservative BBR profile while FEC is on (losses are repaired, so there is no
  need to be as aggressive). A redundancy rate driven by the burstiness of the loss has
  landed: the redundancy follows the peak loss rate of the last second, see 2.5.

## 7. Verified in CI

**quic-go `FEC CI`** (GitHub Actions, all green):

- unit tests: a Cauchy MDS assertion over arbitrary row and member combinations, single
  loss, four packet bursts, unequal packet sizes, a late packet completing an
  underdetermined equation, the idle tail, acknowledgement-only packets being ignored, the
  overhead cap, and repair frame round-trip/truncation/invalid input;
- a regression test for the burst behaviour: after one report of a burst, six clean reports
  follow, and the redundancy has to stay at the level of the burst. Before the fix it had
  decayed to 0.077 by the third of them;
- end-to-end tests over real UDP with loss injection and `-race`: no parity on a clean
  path (`ParityPacketsSent = 0`), recovery at about 12% loss, recovery of bursts of three
  consecutive packets, recovery of bursts of four consecutive packets with the configuration
  QUICX ships with (the 10% cap and the default window), non-zero recovery through the GSO
  send path, and measured overhead within the cap on both endpoints - including a loss rate
  above what the cap can repair, where the transfer still completes on retransmission and
  parity stays within the cap.

**sing-box `FEC CI`** builds real binaries, starts a QUICX server and client on loopback,
fetches 2 MB through SOCKS, and checks that both endpoints logged the
`sliding window scheme` negotiation and that the client logged no FEC activity at all on
the clean path. It then repeats the transfer with `tc netem loss 12%` on `lo` and checks
that the client's statistics line reports a non-zero `rx repaired`, i.e. that the window
scheme reconstructed real losses through sing-box + sing-quic + quic-go.

**Not verified yet**: the window scheme's numbers on a real cross-border mobile path
(`repaired` / `unrecoverable` / `skipped` / throughput over hours). Run `max_group_size` at
a smaller value for a while before changing the default.

**What the fix came from**: 48 minutes of server logs on a real mobile path showed FEC
reconstructing **2 packets** out of 35 MB, while the peer reported loss windows of 15%,
10.5% and 7.5% in the same period, with 0 `unrecoverable` and 0 `dropped`. The repair was
not failing, it never started: the redundancy was driven by the smoothed estimate, a burst
is reported once, and the estimate decayed back to zero while the packets of the burst were
still inside the window, after which no repair row was sent. That does not contradict the
static capacity - the capacity was there, the input driving it was switched off.

## 8. Removal of the block scheme (history)

The three repositories dropped the block scheme in one step and kept only the sliding
window scheme:

- **quic-go**: the group encoder/decoder, `FECScheme` (including `FECConfig.Scheme` and
  `MinGroupSize`), the scheme interface and the `FEC_REPAIR` (0x32) frame with its
  constants are gone. `FECStats` was renamed accordingly: `GroupSize` became `WindowSize`,
  `SkippedGroups` became `SkippedRows`, `ConfiguredOverhead` became `RedundancyRate`, and
  the block-only `Scheme`, `ParityRows` and `ConsideredBytesSent` were removed. The wire
  format of `FEC_WINDOW_REPAIR` (0x34) and `FEC_FEEDBACK` (0x33) is unchanged;
- **sing-quic**: `FECOptions.Scheme` is gone, only the window capability (`0x02`) is
  offered, and the server always confirms the window scheme;
- **sing-box**: the `fec.scheme` option is gone, together with the documentation and CI
  coverage of the removed scheme.

The reason is what a production log review exposed: the block scheme's two weaknesses are
structural, not a matter of parameters. A group that loses more packets than it has parity
rows loses all of them, and the tail of a burst and low rate flows cannot pay for their own
parity. The window scheme covers every capability the block scheme had, so this page
describes only the current implementation; the block scheme's design notes and measurements
were deleted along with the code. The 0x32 frame type is not reused, and the "negotiate
between two schemes" branch is gone; an upgraded endpoint still interoperates with any
version that implements the window scheme, and a block-only version simply runs without
FEC.
