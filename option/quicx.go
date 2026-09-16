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
// FEC is enabled by default: it repairs packets that were lost on the path with parity
// packets, so that the congestion controller doesn't have to react to the loss. Unlike
// Hysteria's "brutal" congestion control, no bandwidth is wasted: the redundancy
// follows the loss rate measured by the peer, and a path without loss doesn't see a
// single parity packet.
//
// Two schemes implement this. The sliding window scheme protects the most recent
// packets with continuously generated repair rows, which recovers bursts of losses and
// protects low rate flows; the block scheme protects a closed group with a fixed number
// of parity rows. Both endpoints negotiate the scheme, so a connection always runs the
// same code.
type QUICXFECOptions struct {
	// Enabled turns packet level FEC on or off. It is enabled by default; set it to
	// false to disable FEC on this endpoint.
	Enabled *bool `json:"enabled,omitempty"`
	// Scheme selects the FEC scheme: "auto" (the default) offers every scheme this
	// build supports and lets the server pick the best one both endpoints implement,
	// "window" or "block" restrict this endpoint to that scheme. A connection on which
	// the two endpoints have no scheme in common runs without FEC.
	Scheme string `json:"scheme,omitempty" enum:"auto,window,block"`
	// MaxOverheadPercent caps the parity traffic as a percentage of the protected
	// traffic. Defaults to 10.
	MaxOverheadPercent int `json:"max_overhead_percent,omitempty"`
	// MaxGroupSize is the maximum number of packets protected by one FEC group (block
	// scheme), or the number of packets one sliding window protects (window scheme).
	// Larger values reduce the relative overhead, but increase the memory and the time
	// until a lost packet can be repaired. Defaults to 32 (block) / 64 (window).
	MaxGroupSize int `json:"max_group_size,omitempty"`
	// MaxParityRows is the maximum number of parity rows per group (block scheme), or
	// the number of repair rows an idle sender emits for the tail of its window (window
	// scheme). 1 repairs a single loss per group, 2 (the default) repairs two.
	MaxParityRows int `json:"max_parity_rows,omitempty"`
}
