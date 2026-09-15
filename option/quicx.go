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

// QUICXFECOptions enables packet level forward error correction for QUICX.
// FEC repairs packets that were lost on the path with parity packets, so that the
// congestion controller doesn't have to react to the loss. Unlike Hysteria's
// "brutal" congestion control, no bandwidth is wasted: the redundancy follows the
// loss rate measured by the peer, and a path without loss doesn't see a single
// parity packet.
type QUICXFECOptions struct {
	// MaxOverheadPercent caps the parity traffic as a percentage of the protected
	// traffic. Defaults to 10.
	MaxOverheadPercent int `json:"max_overhead_percent,omitempty"`
	// MaxGroupSize is the maximum number of packets protected by one FEC group.
	// Larger groups reduce the relative overhead, but increase the time until a lost
	// packet can be repaired. Defaults to 16.
	MaxGroupSize int `json:"max_group_size,omitempty"`
	// MaxParityRows is the maximum number of parity rows per group. 1 (the default)
	// repairs a single loss per group, 2 repairs two losses per group.
	MaxParityRows int `json:"max_parity_rows,omitempty"`
}
