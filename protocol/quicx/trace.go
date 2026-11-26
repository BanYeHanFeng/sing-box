package quicx

import (
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-quic/quicx"
	E "github.com/sagernet/sing/common/exceptions"
)

func buildTrace(options option.QUICXTraceOptions) (quicx.TraceOptions, error) {
	if options.Enabled != nil {
		if !*options.Enabled {
			return quicx.TraceOptions{}, nil
		}
		if options.Path == "" {
			return quicx.TraceOptions{}, E.New("missing trace path")
		}
	}
	return quicx.TraceOptions{
		Path:     options.Path,
		Capacity: options.Capacity,
	}, nil
}
