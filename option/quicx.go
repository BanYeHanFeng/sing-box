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
	Trace       QUICXTraceOptions  `json:"trace,omitempty"`
}

type QUICXUser struct {
	Name     string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
}

type QUICXOutboundOptions struct {
	DialerOptions
	ServerOptions
	Password    string             `json:"password,omitempty"`
	IdleTimeout badoption.Duration `json:"idle_timeout,omitempty"`
	Heartbeat   badoption.Duration `json:"heartbeat,omitempty"`
	MTU         int                `json:"mtu,omitempty"`
	Trace       QUICXTraceOptions  `json:"trace,omitempty"`
	Network     NetworkList        `json:"network,omitempty"`
}

type QUICXTraceOptions struct {
	Enabled  *bool  `json:"enabled,omitempty"`
	Path     string `json:"path,omitempty"`
	Capacity int    `json:"capacity,omitempty"`
}
