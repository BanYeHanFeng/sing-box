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
// retransmission a round trip later. On a path losing 1.5% of its packets, 32+1
// took forwarded datagrams from 1.6% lost to 0.8% and 32+2 to 0.4%; what it costs
// is its redundancy on the wire, which is 3% and 6% of a stream's throughput.
type QUICXFECOptions struct {
	DataShards   int `json:"data_shards,omitempty"`
	ParityShards int `json:"parity_shards,omitempty"`
}

type QUICXOutboundOptions struct {
	DialerOptions
	ServerOptions
	Password    string             `json:"password,omitempty"`
	IdleTimeout badoption.Duration `json:"idle_timeout,omitempty"`
	Heartbeat   badoption.Duration `json:"heartbeat,omitempty"`
	FEC         QUICXFECOptions    `json:"fec,omitempty"`
	Network     NetworkList        `json:"network,omitempty"`
}
