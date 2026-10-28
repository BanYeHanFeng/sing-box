package quicx

import (
	"github.com/sagernet/sing-quic/udpx"
	E "github.com/sagernet/sing/common/exceptions"

	"github.com/sagernet/sing-box/option"
)

// parseFEC turns the redundancy of the configuration into what the transport takes.
//
// The counts are checked here rather than in the transport: a group of one packet
// is a configuration that protects nothing, and more redundancy than members is a
// group sent more times than it has packets. The transport would take either.
func parseFEC(options option.QUICXFECOptions) (udpx.FECOptions, error) {
	if options.DataShards == 0 {
		if options.ParityShards != 0 {
			return udpx.FECOptions{}, E.New("fec: parity_shards of ", options.ParityShards, " without data_shards")
		}
		return udpx.FECOptions{}, nil
	}
	if options.DataShards < 2 {
		return udpx.FECOptions{}, E.New("fec: data_shards of ", options.DataShards)
	}
	if options.ParityShards < 0 {
		return udpx.FECOptions{}, E.New("fec: parity_shards of ", options.ParityShards)
	}
	if options.ParityShards > options.DataShards {
		return udpx.FECOptions{}, E.New("fec: ", options.ParityShards, " parity shards for ", options.DataShards, " data shards")
	}
	return udpx.FECOptions{
		DataShards:   options.DataShards,
		ParityShards: options.ParityShards,
	}, nil
}
