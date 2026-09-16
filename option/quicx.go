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
	// traffic. Defaults to 20.
	MaxOverheadPercent int `json:"max_overhead_percent,omitempty"`
	// MaxGroupSize is the number of packets one sliding window protects. Larger values
	// tolerate longer bursts, but increase the memory FEC uses per connection.
	// Defaults to 128.
	MaxGroupSize int `json:"max_group_size,omitempty"`
	// MaxParityRows is the number of repair rows an idle sender emits for the tail of
	// its window, so that the packets sent last are protected too. Defaults to 2.
	MaxParityRows int `json:"max_parity_rows,omitempty"`
}
