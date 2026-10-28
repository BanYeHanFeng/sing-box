package option

import (
	"github.com/sagernet/sing/common/json/badoption"
)

type QUICXInboundOptions struct {
	ListenOptions
	Users       []QUICXUser        `json:"users,omitempty"`
	UDPTimeout  badoption.Duration `json:"udp_timeout,omitempty"`
	IdleTimeout badoption.Duration `json:"idle_timeout,omitempty"`
	Heartbeat   badoption.Duration `json:"heartbeat,omitempty"`
	MTU         int                `json:"mtu,omitempty"`
	FEC         QUICXFECOptions    `json:"fec,omitempty"`
}

type QUICXUser struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

// QUICXFECOptions is the redundancy this end sends. It covers one direction only,
// so a proxy that wants both directions covered sets it on the inbound and on the
// outbound.
//
// What it is worth depends on what the loss costs. A forwarded UDP packet the
// path drops is gone, and on a path losing 1.5% of its packets, 32+2 took those
// from 1.5% lost to 0.2%. A stream's loss is repaired by a retransmission a round
// trip later, so what redundancy buys there is that round trip: on a path losing
// a tenth of its packets, 32+2 cut the p95 of a 1 MB transfer by a third and left
// a full-speed transfer within a few percent of what it was without any parity,
// at a sixteenth of the traffic and a fifth more CPU. Below a few percent of loss
// the parity is a cost with nothing to buy: what a group can repair is only ever
// as much as the path drops inside it.
type QUICXFECOptions struct {
	Enabled      bool `json:"enabled,omitempty"`
	DataShards   int  `json:"data_shards,omitempty"`
	ParityShards int  `json:"parity_shards,omitempty"`
}

type QUICXOutboundOptions struct {
	DialerOptions
	ServerOptions
	Password    string             `json:"password,omitempty"`
	IdleTimeout badoption.Duration `json:"idle_timeout,omitempty"`
	Heartbeat   badoption.Duration `json:"heartbeat,omitempty"`
	MTU         int                `json:"mtu,omitempty"`
	FEC         QUICXFECOptions    `json:"fec,omitempty"`
	Network     NetworkList        `json:"network,omitempty"`
}
