# QUICX FEC: surviving packet loss with erasure codes

This page answers one question: can QUICX use a "hard disk style" checksum (RAID-like
erasure coding, i.e. forward error correction) to survive packet loss, instead of
sending blindly at a fixed rate the way Hysteria2's `brutal` congestion control does -
and without wasting bandwidth?

The answer is **yes**. It is implemented and verified in CI over a lossy link. The core
idea is adaptive redundancy: no parity packet at all is sent on a path that doesn't lose
packets, and on a lossy path the amount of redundancy follows the measured loss rate,
capped by a configurable bound.

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
3. the **sender** smooths the reports with an EWMA (alpha = 0.3), targets
   `1.5 x loss rate`, clamped to `[1/max_group_size, max_overhead_percent]`, and derives
   the group size `k` from it;
4. below 0.2% loss FEC is **completely off**: `k = 0`, not a single parity packet
   (verified in CI: `ParityPacketsSent = 0` on a clean path);
5. a partial group is protected after 2ms of idle time, so the last packets of a burst
   aren't left unprotected.

## 5. Overhead

Redundancy is `m / k`. With a single parity row:

| Measured loss p | Target redundancy 1.5p | Group k (max 16) | Actual redundancy | P(>= 2 losses in group) |
| --- | --- | --- | --- | --- |
| < 0.2% | - | off | 0 | - |
| 1% | 1.5% | 16 | 6.25% | 1.1% |
| 2% | 3% | 16 | 6.25% | 4.0% |
| 5% | 7.5% | 13 | 7.7% | 13.5% |
| 10% | 10% (capped) | 10 | 10% | 26.4% |
| 20% | 10% (capped) | 10 | 10% | 62.4% |

Notes:

- redundancy comes in steps of `1/k`, so at very low loss rates the actual overhead is a
  few times the loss rate (6.25% vs 1%). `max_group_size: 32` lowers the floor to 3.1%,
  at the cost of a longer repair delay;
- above roughly 10-15% loss a single parity row leaves many groups unrepaired;
  `max_parity_rows: 2` (Reed-Solomon) repairs two losses per group, but it is only enabled
  when `max_overhead_percent >= 2/max_group_size`;
- FEC also reduces the maximum protected packet size by 40-130 bytes (depending on
  `max_group_size`). This overhead exists only while FEC is active, i.e. only on lossy
  paths.

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

Both sides log a debug line once FEC is negotiated:

```
QUICX FEC enabled (client, max overhead 10%, max group 16)
```

While FEC is enabled, a statistics line is written every 10 seconds (windows without
any FEC activity are skipped):

```
QUICX FEC: path loss 3.4%, group 13 (overhead 7.7%), repaired 128, unrecoverable 9,
  parity 96 sent / 91 received, protected 1200 packets (14.2 KB parity data)
```

- The line is written at **debug** level, except when packets were actually repaired
  (`repaired`/`unrecoverable` > 0): those windows are logged at **info** level, at most
  once a minute per connection, so a lossy server doesn't flood its log. In other words,
  the default `"log": {"level": "info"}` is enough to see whether FEC is doing
  something; use `"log": {"level": "debug"}` to also see the idle (zero redundancy)
  windows.
- Fields: `path loss` is the loss rate the peer measured from packet number gaps
  (including packets FEC repaired, i.e. the real path quality); `group`/`overhead` is the
  current group size and redundancy, `idle` means no loss and therefore no redundancy;
  `repaired` is the number of packets reconstructed from parity in this window;
  `unrecoverable` counts packets parity couldn't repair (they fall back to QUIC
  retransmission); `parity sent / received` and `parity data` show the redundancy cost.

## 9. Configuration

```json
{
  "type": "quicx",
  "fec": {
    "max_overhead_percent": 10,
    "max_group_size": 16,
    "max_parity_rows": 1
  }
}
```

FEC has to be enabled on both sides; the client announces support in its authentication
request and only enables FEC once the server confirmed it, so enabling it on one side
never wastes bandwidth.
