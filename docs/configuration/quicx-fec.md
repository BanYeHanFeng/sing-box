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

## 7. Configuration

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
