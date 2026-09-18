package option

import (
	"github.com/sagernet/sing/common/json/badoption"
)

type QUICXInboundOptions struct {
	ListenOptions
	Users             []QUICXUser        `json:"users,omitempty"`
	AuthTimeout       badoption.Duration `json:"auth_timeout,omitempty"`
	Heartbeat         badoption.Duration `json:"heartbeat,omitempty"`
	AuthFailurePolicy string             `json:"auth_failure_policy,omitempty" enum:"h3_close,silent_drop"`
	BBRProfile        string             `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	FEC               *QUICXFECOptions   `json:"fec,omitempty"`
	InboundTLSOptionsContainer
	QUICOptions
}

type QUICXUser struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

type QUICXOutboundOptions struct {
	DialerOptions
	ServerOptions
	Password   string             `json:"password,omitempty"`
	Heartbeat  badoption.Duration `json:"heartbeat,omitempty"`
	BBRProfile string             `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	FEC        *QUICXFECOptions   `json:"fec,omitempty"`
	Network    NetworkList        `json:"network,omitempty"`
	OutboundTLSOptionsContainer
	QUICOptions
}

// QUICXFECOptions configures packet level forward error correction for QUICX.
//
// FEC is enabled by default: it repairs packets that were lost on the path with repair
// packets, so that the congestion controller doesn't have to react to the loss. Unlike
// Hysteria's "brutal" congestion control, no bandwidth is wasted: the redundancy
// follows the loss rate measured by the peer, and a path without loss doesn't see a
// single repair packet.
//
// The sliding window scheme implements this: the most recent packets are protected by
// continuously generated repair rows, so a burst of losses is reconstructed row by row
// and low rate flows are protected too. Both endpoints negotiate FEC, so a connection
// always runs with FEC on both ends or on neither.
type QUICXFECOptions struct {
	// Enabled turns packet level FEC on or off. It is enabled by default; set it to
	// false to disable FEC on this endpoint.
	Enabled *bool `json:"enabled,omitempty"`
	// MaxOverheadPercent caps the repair traffic as a percentage of the protected
	// traffic. Defaults to 30: with the 1.5x-loss safety factor, the reactive rate
	// stays uncapped through about 20% measured random loss, the knee measured by
	// the same-packet QUICX FEC / HY2 CI comparison.
	MaxOverheadPercent int `json:"max_overhead_percent,omitempty"`
	// MaxGroupSize is the number of packets one sliding window protects. Larger values
	// tolerate longer bursts, but increase the memory FEC uses per connection.
	// Defaults to 128.
	MaxGroupSize int `json:"max_group_size,omitempty"`
	// MaxParityRows is the number of repair rows an idle sender emits for the tail of
	// its window, so that the packets sent last are protected too. Defaults to 2.
	MaxParityRows int `json:"max_parity_rows,omitempty"`
	// BaselineRedundancyPercent keeps a small fixed redundancy on the wire even while
	// the path looks lossless, so the first burst doesn't have to wait for the peer's
	// feedback. Bounded by MaxOverheadPercent. Defaults to 5, a knee for links that
	// occasionally lose around 8% of packets in a burst; set it explicitly to 0 to
	// return to a purely reactive path.
	BaselineRedundancyPercent *int `json:"baseline_redundancy_percent,omitempty"`
	// RecoveredPacketFeedback reports packets this endpoint reconstructed with FEC back
	// to the sender, so its congestion controller sees the loss without retransmitting
	// the packet (RFC 9265, known-lossy-path exception). Enabled by default; both ends
	// have to understand the FEC_RECOVERED frame, so set it to false only when
	// the peer does not support it.
	RecoveredPacketFeedback *bool `json:"recovered_packet_feedback,omitempty"`
	// AdaptiveWindow lets the sender size its working window from the measured RTT and
	// packet rate, between a lower bound and MaxGroupSize. It changes no wire format and
	// is enabled by default in this phase 2 build; set it explicitly to false to pin the
	// window at MaxGroupSize. A pointer distinguishes "not configured" from false.
	AdaptiveWindow *bool `json:"adaptive_window,omitempty"`
	// MultiWindow enables the multi-window scheme (capability 0x04), which splits the
	// protected stream into independent sub-windows assigned by packet number modulo
	// the count. It is enabled by default in this phase 2 build; set it explicitly to
	// false for a fixed single window. The effective window becomes
	// MultiWindowCount * MaxGroupSize.
	MultiWindow *bool `json:"multi_window,omitempty"`
	// MultiWindowCount is the number of sub-windows when MultiWindow is enabled.
	// Defaults to 2 and is capped at 4.
	MultiWindowCount int `json:"multi_window_count,omitempty"`
	// RepairBurstRowsPerLoss is the number of repair rows scheduled per lost packet
	// after a feedback report. It is an internal experiment value for the phase 2
	// field measurements; zero keeps the default of 2.0.
	RepairBurstRowsPerLoss float64 `json:"repair_burst_rows_per_loss,omitempty"`
}
