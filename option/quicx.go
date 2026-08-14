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
	InboundTLSOptionsContainer
	QUICOptions
}

type QUICXUser struct {
	Name     string `json:"name,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
}

type QUICXOutboundOptions struct {
	DialerOptions
	ServerOptions
	UUID      string             `json:"uuid,omitempty"`
	Password  string             `json:"password,omitempty"`
	Heartbeat badoption.Duration `json:"heartbeat,omitempty"`
	Network   NetworkList        `json:"network,omitempty"`
	OutboundTLSOptionsContainer
	QUICOptions
}
