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
// It is worth its cost only for what the transport cannot send again: a forwarded
// UDP packet the path drops is gone, while a stream's loss is repaired by a
// retransmission a round trip later. On a path losing 1.5% of its packets, 32+2
// took forwarded datagrams from 1.5% lost to 0.2%; what it costs is its redundancy
// on the wire, which is 6% of a stream's throughput. Turning it on for streams does
// not help: a lost packet there costs one round trip rather than bandwidth, and at
// the rate a proxy actually sends a group closes before it fills, so the parity is
// a seventh of the traffic rather than a thirty-second.
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
