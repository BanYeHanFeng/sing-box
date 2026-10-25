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
	BBRProfile  string             `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	FEC         QUICXFECOptions    `json:"fec,omitempty"`
}

type QUICXUser struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

// QUICXFECOptions is the redundancy this end sends. It covers one direction only,
// so a proxy that wants both directions covered sets it on the inbound and on the
// outbound.
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
	BBRProfile  string             `json:"bbr_profile,omitempty" enum:"standard,conservative,aggressive"`
	FEC         QUICXFECOptions    `json:"fec,omitempty"`
	Network     NetworkList        `json:"network,omitempty"`
}
