# QUICX FEC 二期任务书（待实现）

> 本文是交给后续实现 AI / 工程师的任务书，**只描述方案、协议、代码落点与验收标准，不包含实现**。
> 一期已经落地的内容见 [QUICX FEC](quicx-fec.zh.md)：累计去重丢包证据、丢包触发的修复突发、
> 127 个 Cauchy 行基、127 条待解方程上限、256 KB 突发额度。
>
> 本文中的配置名、字段名、帧类型编号和 capability bit 都是**建议值**；实现时若与既有风格或线上
> 兼容性冲突，可以在不改变语义的前提下调整，但要同步更新本文与 `quicx-fec.zh.md`。

---

## 1. 目标与范围

一期重构解决了“有足够校验量时，突发确实能被解码器吃下”的问题：日志中的 `repaired` 明显上升，
但真实弱网仍有四类场景会退回到 QUIC 重传：

1. **高 pps / 高 RTT / 大 BDP**：单窗口最大 128 包，在数万 pps 下只覆盖几毫秒到几十毫秒，
   反馈到达前丢失包已经离开发送端窗口；
2. **反馈粒度太粗**：`FEC_FEEDBACK` 只有累计计数，发送端不知道具体哪些包仍然缺失，只能用
   `deltaLost` 估算补发行数，也无法判断这些包是否还在窗口里；
3. **窗口大小固定**：快速干净链路被 128 包窗口放大内存/RTT 估算误差，慢速易丢链路又需要更长
   时间覆盖；
4. **保底冗余与拥塞策略缺少真实链路数据**：默认 5% 是否应改为 8%~10%、FEC 开启时 BBR 是否应
   更保守，需要长期日志 A/B 才能定论。

二期目标：在上述四类场景中继续提高 `repaired / (repaired + unrecoverable)`，同时**不突破
`max_overhead_percent` 的长期字节预算**，不破坏旧端点互通。

非目标：

- 不把 QUICX 改成 Hysteria2 `brutal` 那样的固定高速率方案；
- 不替换已确认的 Cauchy / 滑动窗口主体；
- 不在二期做多路径、前向安全、或加密层改动；
- 不在本文档中直接写生产代码。

---

## 2. 当前基线（实现前置信息）

### 2.1 代码落点

**quic-go**（`github.com/sagernet/quic-go`，分支 `BanYeHanFeng-dev`）

| 文件 | 关键内容 |
| --- | --- |
| `fec.go` | `FECConfig` / `FECStats` / `enableFEC` / `fecLossTracker` / `handleFECFrame` / `handleFECRecoveredFrame` |
| `fec_window.go` | 编码器、解码器、`scheduleRepairBurst`、`emitBurstRow`、`pendingFrame`、`pendingFeedback`、`fecWindowCoefficient` |
| `internal/wire/fec_frame.go` | `0x33 FEC_FEEDBACK`、`0x34 FEC_WINDOW_REPAIR`、`0x36 FEC_RECOVERED` 的编解码与上限 |
| `fec_window_test.go` / `fec_integration_test.go` | 编解码、突发、尾部、额度、周期丢包仿真测试 |
| `connection.go` | `fecRecordSentPacket` / `fecRecordReceivedPacket` / `maybeSendFECPackets` / send loop 挂点 |

**sing-quic**（`github.com/sagernet/sing-quic`，分支 `BanYeHanFeng-dev`）

| 文件 | 关键内容 |
| --- | --- |
| `quicx/fec.go` | `FECOptions`、能力位 `fecCapabilityWindow = 0x02`、`startFEC` / `notifyFECAccept`、`formatFECStats` |
| `quicx/*.go` | QUICX 握手、鉴权请求尾部能力位读取、服务端确认流 |

**sing-box**（`github.com/BanYeHanFeng/sing-box`，分支 `BanYeHanFeng-dev`）

| 文件 | 关键内容 |
| --- | --- |
| `option/quicx.go` | `QUICXFECOptions` 配置字段 |
| `protocol/quicx/inbound.go` / `outbound.go` | `buildFECOptions()`，默认 `baseline_redundancy_percent=5`、`max_overhead_percent=30` |
| `docs/configuration/quicx-fec.zh.md` | 用户文档、日志字段、容量表 |
| `.github/workflows/fec-ci.yml` | 使用发布流水线方式构建真实 sing-box 并做端到端丢包验证 |

### 2.2 当前协议

```text
0x33 FEC_FEEDBACK:
  received | lost_evidence | recovered | failed | parity_received

0x34 FEC_WINDOW_REPAIR:
  row | first_packet_number | span | member_bitmap | member_count
      | parity_length | length_parity[2] | parity_data

0x36 FEC_RECOVERED:
  count | first_packet_number | (pn_delta | wire_length) * count
```

- `lost_evidence` 一期已改为：包号空洞 + 校验行揭示的缺失包，去重后累计，只增不减。
- capability bit 目前只有 `0x02`，服务端确认也必须回 `0x02`。
- 一期新增的修复突发完全复用 `0x34`，不涉及新帧类型；这是线上兼容的关键。

### 2.3 当前关键常量

```text
MaxFECWindowSize       = 128     // 单个 0x34 行最多保护的成员数
MaxFECWindowSpan       = 512     // member bitmap 最大包号跨度
fecWindowCauchyRows    = 127     // 独立行基上限
fecWindowMaxPendingRows= 127     // 解码器待解方程上限
fecWindowCreditLimit   = 256 KB  // 突发可动用的累计额度上限
fecWindowRepairBurst   = 2 行 / 丢包，单次最多 127 行
```

---

## 3. 二期总方案

建议按风险 / 收益拆成四个可独立合入的阶段：

| 阶段 | 方案 | 是否需要新 capability | 主要收益 | 主要成本 |
| --- | --- | --- | --- | --- |
| P1 | **显式缺失包区间反馈** | 是（`0x08`） | 发送端精确知道哪些包还缺、哪些还能修 | 新帧格式、反馈大小控制 |
| P2 | **RTT 自适应窗口** | 否 | 高 RTT/低速率链路减少无效窗口内存，快速链路降低反馈后覆盖时间 | 编码器动态窗口、滞回策略 |
| P3 | **多子窗口交错编码** | 是（`0x04`） | 有效窗口提升到 `k*128`，对抗高 pps 下窗口过期 | 多窗口状态机、内存、位图/行号语义 |
| P4 | **保底冗余 / BBR 策略实测定标** | 否（配置/日志） | 用真实链路数据确定 baseline 与拥塞策略 | 长时线上测试、统计口径 |

实施顺序建议：**P1 先做协议扩展和反馈基础设施；P2 可并行；P3 依赖 P1 的缺失区间反馈来分配各子窗口
的补发行；P4 在 P1~P3 任一阶段可用后开始收集数据。**

---

## 4. P1：显式缺失包区间反馈（优先级最高）

### 4.1 要解决的问题

一期 `LostPackets` 是累计去重后的总量，发送端只知道“新丢了 N 个包”，但：

- 这 N 个包可能已经离开发送端窗口，补发了也修不回来；
- 也可能其中有 ACK-only、纯流控包、校验包，根本不需要 FEC；
- 发送端无法按“仍可修复的缺失包数”精确补发，只能靠 `2 行/丢包` 的保守系数；
- 尾部突发虽然已经能通过 `countLoss` 上报，但发送端收到上报后仍不知道丢失包号是否还在
  `encoder.members` 里。

### 4.2 协议设计

新增 capability bit：

```go
const (
    fecCapabilityWindow        = 0x02 // 一期协议
    fecCapabilityMultiWindow   = 0x04 // 三期预留（本任务书 P3）
    fecCapabilityMissingRanges = 0x08 // 本阶段
)
```

- 新客户端发 `0x02 | 0x08`（若同时声明多窗口则加 `0x04`）；
- 新服务端确认自己支持且双方都有的最高能力组合；
- 旧服务端会 `& 0x02` 后确认 `0x02`，新客户端自动退回一期行为；
- 确认必须逐 bit 校验，不得接受客户端没声明、或本地未实现的 bit。

新增帧 `0x37 FEC_FEEDBACK_V2`（建议编号，不能与 `0x36 FEC_RECOVERED` 冲突）：

```text
0x37 FEC_FEEDBACK_V2:
  received        varint   // 累计收到 1-RTT 包数
  lost_evidence   varint   // 一期累计去重丢包证据（兼容发送端估算器）
  recovered       varint
  failed          varint
  parity_received varint
  missing_count   varint   // 本帧携带的“当前缺失包区间数”
  ranges...                // 每项：first_packet_number varint, count varint
```

约束：

- `ranges` 只包含**当前仍标记为 missing 的被保护包**，按包号升序；
- 一帧最多携带 `fecWindowFeedbackMaxRanges` 个区间 / `fecWindowFeedbackMaxMissing` 个包号，
  例如 32 个区间或 128 个包号，先到先截断；截断策略优先保留**包号较大（较新）**的缺失，
  因为旧包更可能已失效；
- 必须保证整个 `FEC_FEEDBACK_V2` 能放进一个 ACK-only 短包；构建时用 `wire.FECFeedbackV2Frame.Length()`
  校验，放不下就减少区间；
- 累计计数器 `received/lost_evidence/...` 与 `0x33` 保持同样语义，便于发送端共用估算器；
- 反馈帧自身丢失：累计计数器会补偿；缺失区间下一帧重发当前 missing，不依赖可靠传输。

兼容：

- 未协商 `0x08` 的旧端点只收发 `0x33`；
- 新端点两种帧都能解析，发送哪种由协商结果决定；
- `wire.IsFECFrameType`、`handleFrame` 的 `case`、`frame_format` / 解析入口全部要补。

### 4.3 解码端实现

文件：`quic-go/fec_window.go`、`internal/wire/fec_frame.go`。

1. `fecWindowDecoder` 已有 `missing map[PacketNumber]time`，直接作为区间数据源；
2. 在 `pendingFeedback` 中，根据 `d.config.ExtendedFeedback`（来自 capability 协商）选择构建
   `FECFeedbackV2Frame`；
3. 增加 `missingRanges()`：
   - 遍历 map，按包号排序；
   - 合并连续/近连续包号（建议 gap <= 3 也合并，减少区间数）；
   - 按“优先新包号”截断；
4. 增加 `reportedMissing` 快照，避免每 20ms 重复发送完全相同的区间；若 missing 集合没变化，
   允许只发累计计数器（`missing_count=0`）；
5. 注意：`missing` 中既有被保护包，也有解码器还没收到但已由校验行揭示的包；只发送
   `markProtected` 过的包号清单，避免把未保护包也塞进 ranges。

### 4.4 发送端实现

文件：`quic-go/fec_window.go`、`quic-go/fec.go`、`sing-quic/quicx/fec.go`。

1. `FECConfig` 增加：
   ```go
   ExtendedFeedback bool // 协商结果由 sing-quic 设置
   ```
2. `fecWindowEncoder` 增加处理 `*wire.FECFeedbackV2Frame` 的分支；
3. 对每个 missing range / packet number：
   - 查当前 `encoder.members` 是否仍包含该 pn；
   - 仍包含：计入“可修缺失数”，按一期 `2 行/丢包` 排入 burst；
   - 已不包含：不计入当前补发，但保留累计 `lost_evidence` 用于估算器降速率；
4. burst 总量仍受 `fecWindowCauchyRows`、`fecWindowCreditLimit`、`max_overhead_percent` 约束；
5. 若可修缺失数明显小于累计 deltaLost，说明反馈确实精确了，不能因此把稳态 rate 降到 0；
   估算器继续使用 `lost_evidence` 的累计差值。
6. `FECStats` 增加：
   - `RepairBurstRowsScheduled`
   - `RepairBurstRowsSkippedNoWindow`
   - `MissingRangesReceived`
   - `MissingPacketsInWindow`
7. `formatFECStats` 在 `FECStats` 后追加：
   - `missing-ranges N`（收到/上报区间数）
   - `burst ...` 可细分为 `burst ... (M no-window)`。

### 4.5 测试

**单测**（quic-go）：

- `TestFECFeedbackV2RoundTrip`
- `TestFECFeedbackV2TruncatedAndMalformed`
- `TestFECMissingRangesMergeAndCap`
- `TestFECFeedbackV2FitsIntoAckOnlyPacket`
- `TestFECV2SchedulesBurstOnlyForPacketsStillInWindow`
- `TestFECV2CompatibleWithV1CounterEstimation`

**集成**（真实 UDP + 丢包注入）：

- 构造 128 包窗口内 60 包突发，反馈返回精确 ranges，断言 `repaired` 达到可修缺失的
  `>= 90%`；
- 构造反馈前丢失包全部离开窗口（高 pps + 长反馈延迟），断言不出现无效 burst，且速率
  降至低冗余而不是卡在上限；
- 约 12% 随机丢包、每 32 包 4 连突发，payload 校验一致；
- 旧端点兼容：客户端/服务端一方不支持 `0x08` 时连接正常，行为退回 `0x33`。

**sing-box 端到端**：

- `tc netem` 或 UDP relay 注入“低平均丢包 + 每 N 包大突发”；
- 客户端 `rx repaired`、服务端 `rx repaired` 明显上升，`unrecoverable` 下降；
- 旧版客户端/新版服务端、新版客户端/旧版服务端都通过基本传输。

### 4.6 验收标准

- 可修窗口内突发（<=127 行可表达）修复率 `repaired / (repaired+unrecoverable) >= 90%`；
- 反馈包单帧不超过 datagram；恶意 128 个区间不会导致内存/CPU 无界增长；
- 不支持新 bit 的旧端点退化为一期能力，无连接失败；
- 长期 `measured parity bytes / protected bytes <= max_overhead_percent` 不被突破。

---

## 5. P2：RTT 自适应窗口

### 5.1 要解决的问题

`MaxGroupSize` 固定 128 包。快速链路（数千~数万 pps）只需要很小的窗口就能积累足够修复行，
但 128 包会带来更大的状态和更长的反馈队列；慢速链路需要更长的时间覆盖，固定 128 又不够。
一期已经把协议上限和行基用满，P2 先在不改变协议的前提下，让**实际工作窗口**
在 `[64, MaxFECWindowSize]` 间随 RTT / 发包间隔变化。

### 5.2 算法建议

发送端已有 `averageInterval`（EWMA 包间隔）与 `windowSize`。增加：

```text
effectiveWindow = clamp(round(rtt * packetRate * fecWindowRuntimeScale), min=64, max=config.MaxGroupSize)
packetRate      = 1 / averageInterval
fecWindowRuntimeScale 建议 1.0~2.0，先用 1.5
```

- 用 `c.rttStats.SmoothedRTT()`；没有 RTT 时保持默认 128；
- 调整必须带**滞回**：目标窗口变化超过 20% 且持续 >= 1 秒才切，避免抖动；
- 调小窗口时只影响新行，旧行仍按原成员描述；解码端 `cacheSize` 本来就按配置最大值分配，
  不需要跟随变小；
- `holdDuration()` 当前按 `windowSize * averageInterval` 计算，动态窗口后自动跟随；
- `fecWindowMaxCoverageRows` 约束、`rowCredit`、`credit`、burst 上限都用运行窗口重新计算；
- 统计行增加 `window 83/128 pkts (rtt-adaptive)`，便于线上确认。
- `MaxGroupSize` 仍作为协商时双方支持的协议上限，不把动态值写进 capability。

### 5.3 配置

建议 sing-box 增加（可选，默认 `false` 保持一期行为）：

```json
"fec": {
  "adaptive_window": false
}
```

- 也可先只做服务端/客户端内部实验选项，稳定后再暴露文档。

### 5.4 测试与验收

- 单测：`TestFECAdaptiveWindowFollowsRTTAndRate`、`TestFECAdaptiveWindowHysteresis`；
- 集成：同一台机器用 `tc netem delay` + 不同 packet pacing，验证快速链路窗口下降、
  高 RTT 链路窗口上升；
- 验收：干净链路 `ParityPacketsSent` 仍为 0（当 baseline=0）；丢包链路 payload 一致；
  P99 内存不高于固定 128 包方案。

---

## 6. P3：多子窗口交错编码

### 6.1 要解决的问题

即使 P1/P2 完成，单个 `0x34` 行仍最多保护 128 个包。高 pps / 高 RTT / 长时间 burst 下，
从“收到丢包反馈”到“修复行发出”的时间内，丢失包可能已经滚出唯一窗口。

### 6.2 方案

把当前单个 `fecWindowEncoder` / `fecWindowDecoder` 变成 `k` 个独立子窗口：

- `k` 建议 2 或 4，由 capability 协商或配置决定；
- 数据包按 `packet_number % k` 或按“到达轮次”分配到子窗口；推荐 `pn % k`，实现简单、
  解码端可预测；
- 每个子窗口仍是 128 包、127 个行基；行基可以复用，也可以每个子窗口用不同 row base 偏移，
  推荐复用，因为子窗口成员集合不重叠；
- 有效窗口 = `k * 128` 包；突发容量近似 `k` 倍稳态 + 各子窗口修复突发；
- 每个子窗口独立维护 `members`、`rowSeq`、`rowCredit`、`credit share`、burst 行数。

### 6.3 协议

复用一期协商框架，新增：

```go
fecCapabilityMultiWindow = 0x04
```

新增帧 `0x38 FEC_WINDOW_REPAIR_MULTI`（建议编号）：

```text
0x38:
  window_id       varint  // 0..k-1
  row             varint
  first_packet_number
  span
  member_bitmap
  member_count
  parity_length
  length_parity[2]
  parity_data
```

- `window_id` 是新增字段，旧 `0x34` 不含；新帧只在双方确认 `0x04` 后使用；
- 解析时 `window_id` 必须 `< negotiated_window_count`，否则丢弃且不计可恢复；
- `member_count/span` 复用 `0x34` 上限，避免单帧变大。

### 6.4 反馈与调度

- `0x33` 的累计计数器仍全局；一期 `0x34` 的 `LostPackets` 估算器继续保持原语义；
- P1 的 `0x37 FEC_FEEDBACK_V2` 需要按窗口归属反馈 missing ranges：
  - 解码端 `pendingFeedback` 构建 lost 时根据 `pn % k` 拆到不同子窗口；
  - 发送端按窗口调度 burst，只向仍包含该窗口丢失包的子窗口补行；
- 若某个子窗口的命中率长期显著低于其他窗口，日志输出 `window_id` 以便定位路径/哈希问题。

### 6.5 兼容与资源上限

- 旧 `0x34` 与新 `0x38` 可同时被同一连接支持，但协商后只发一种；
- 未协商 `0x04` 时退回单窗口，行为与一期完全一致；
- 每连接内存约增加 `(k-1) * (2 * W * MTU + 状态)`；`k=4`、MTU=1500 时约 `(3*2*128*1500)` ≈ 1.1 MB，
  必须在 Task 中明确写在文档里并做 profiler 验证；
- 总字节额度 `fecWindowCreditLimit` 在所有子窗口间共享，避免每个子窗口各持一份 256 KB；
- 子窗口数量、有效窗口大小写入 `FECStats` 与统计行。

### 6.6 测试与验收

- 单测：`TestFECMultiWindowAssignment`、`TestFECMultiWindowRepairFrameRoundTrip`、
  `TestFECMultiWindowBudgetShared`；
- 集成：`netem` 下 10k~50k pps 连续丢包突发，单窗口版本失败、多窗口版本恢复率提升；
- 验收：
  - 有效窗口按 `k` 增长（统计可观测）；
  - 每个子窗口单独丢包、跨子窗口混合丢包都能修；
  - 长期 overhead cap 仍不被突破；
  - 关闭 capability 时连接与一期完全等价。

---

## 7. P4：保底冗余与 BBR 策略实测定标

### 7.1 要回答的问题

- `baseline_redundancy_percent` 默认 5 是否不够？8~10 是否会显著抬高干净链路成本？
- FEC 开启后，丢包被修复，BBR 是否应换更保守 profile，避免校验流量和重传共同推高排队？
- `repair_burst_rows_per_loss = 2` 在真实链路是否过高/偏低？
- `max_overhead_percent = 30` 是否应按链路 RTT 分档？

### 7.2 需要先补的观测字段

在现有 `FECStats` 基础上建议增加：

- `RepairBurstRowsScheduled`
- `RepairBurstRowsSkippedBudget`
- `RepairBurstRowsSkippedNoWindow`
- `MissingRangesReceived`
- `MissingPacketsRepairable`
- `FeedbackLatency`（从 receiver missing 首次进表到 sender 收到反馈的估算）
- `ProtectedPacketRate`（pps）与 `PeerLossPeak`

`formatFECStats` 增加上述字段的可读输出，或额外输出 JSONL 日志供采集。

### 7.3 实验矩阵

| 变量 | 档位 |
| --- | --- |
| `baseline_redundancy_percent` | 0 / 5 / 8 / 10 |
| `repair_burst_rows_per_loss` | 1.5 / 2 / 2.5（内部实验参数） |
| `max_overhead_percent` | 30 / 40 / 50（只用于研究上限膝点） |
| BBR profile | standard / conservative |

每档至少覆盖：干净链路、约 3% 稀疏突发、约 10% 随机丢包、约 10% 周期性 60 包突发。

### 7.4 指标与验收

- `repair_hit = repaired / (repaired + unrecoverable)`；
- `attempt_coverage = (repaired + unrecoverable) / estimated_protected_lost`；
- `overhead = cumulative parity bytes / cumulative protected bytes`；
- goodput、RTT inflation、重传字节；
- 目标：
  - 对于突发 <= 32 包且平均丢包 <= 5%，`repair_hit >= 85%`；
  - 打开 FEC 相对关闭 FEC 的额外带宽不超过 `max_overhead_percent`；
  - 干净链路若 baseline=0，仍然 `ParityPacketsSent = 0`。

### 7.5 交付物

- 一份真实链路测试报告（原始日志 + 汇总表 + 结论）；
- 默认值变更 PR（如有）；
- 若调整 BBR，改为由 FEC 开启状态自动选择 profile，并保留显式配置覆盖。

---

## 8. 跨仓库实施清单（给实现 AI）

### 8.1 quic-go

- [ ] `FECConfig` / `FECStats` 增加 P1/P2/P3 所需字段；
- [ ] `internal/wire/fec_frame.go` 增加 `0x37` / `0x38` 帧类型、解析、长度、上限；
- [ ] `fecWindowDecoder` 生成缺失区间；
- [ ] `fecWindowEncoder` 支持 P1 精确 burst、P2 动态窗口、P3 多子窗口；
- [ ] `connection.go` 的 `handleFECFrame` / `pendingFrame` / `maybeSendFECPackets` 接入新帧；
- [ ] 单测、集成测试、race、基准；
- [ ] `fec-ci.yml` 增加新测试和阈值。

### 8.2 sing-quic

- [ ] `FECOptions` 增加 `ExtendedFeedback`、`MultiWindow`、`AdaptiveWindow`（含默认关闭的兼容策略）；
- [ ] capability `fecCapabilityWindow | fecCapabilityMissingRanges | fecCapabilityMultiWindow`；
- [ ] `startFEC` / `notifyFECAccept` 逐 bit 协商，旧服务端自动回退；
- [ ] `formatFECStats` 输出新字段；
- [ ] 协商/统计单测。

### 8.3 sing-box

- [ ] `option/quicx.go` 增加 P2/P3/P4 配置项（建议默认关闭，等二期稳定后再评估默认值）；
- [ ] `protocol/quicx/inbound.go` / `outbound.go` 透传；
- [ ] `docs/schema.json`、中英文 `quicx-fec` 文档同步；
- [ ] FEC CI 增加“新版/旧版混合协商”和“P1/P3 突发”用例；
- [ ] 保持 P1 之前版本可正常构建和运行。

---

## 9. 建议的 PR 拆分

1. **PR-A**：P1 协议与反馈基础设施（`0x37`、capability `0x08`、单测）；
2. **PR-B**：P1 发送端精确 burst（依赖 PR-A）；
3. **PR-C**：P2 RTT 自适应窗口（独立，无协议变更）；
4. **PR-D**：P3 多子窗口（依赖 PR-A/PR-B，建议默认关闭）；
5. **PR-E**：P4 观测字段与实验开关（可以最早合，用于收集数据）；
6. **PR-F**：文档、默认值与 CI 验收更新。

每个 PR 必须包含：单元测试、至少一个集成/端到端回归、统计字段、向后兼容说明、以及
临时 VPS / APB 或 GitHub FEC CI 的完整测试证据。

---

## 10. 风险与回滚

| 风险 | 缓解 |
| --- | --- |
| 新帧类型与旧端点冲突 | capability 协商后才发送；未协商时只用 `0x33/0x34` |
| 缺失区间反馈过大导致 ACK 包超 MTU | `Length()` 预算检查，超限截断区间数 |
| 多子窗口内存增长 | `k <= 4`、共享额度、默认关闭、上线前 profiler |
| 动态窗口抖动 | 滞回 + 最小调整间隔 + 仅影响后续行 |
| 精确 burst 可能更浪费在中继场景 | 仍受 `max_overhead_percent` 约束；保留 `skipped` 统计 |
| 默认值变更导致线上带宽上升 | P4 先出报告，后改默认值；提供显式配置覆盖 |

回滚策略：

- 协议扩展全部由 capability bit 开关控制；
- 新版客户端遇到旧服务端、旧版客户端遇到新服务端都必须自动降级；
- 出问题时可通过配置显式关闭 P1/P2/P3，不需要回滚二进制；
- 默认值变更单独 PR、单独发布说明，可快速回退。

---

## 11. 验收总表

- [ ] P1：窗口内可修突发修复率 >= 90%，旧端点无感降级；
- [ ] P2：RTT/速率变化时窗口按滞回调整，干净链路零冗余不回归；
- [ ] P3：有效窗口提升到 `k*128`，高 pps 突发修复率相对单窗口显著提升，内存上限有报告；
- [ ] P4：完成实验矩阵与真实链路报告，给出基线默认值结论；
- [ ] 三仓库 FEC CI 全绿；
- [ ] `measured overhead <= max_overhead_percent` 仍成立；
- [ ] 线上旧版本互通矩阵通过：新-新、新-旧、旧-新。
