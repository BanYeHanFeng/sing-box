package quicx

import (
	"github.com/sagernet/sing-quic/udpx"
	E "github.com/sagernet/sing/common/exceptions"

	"github.com/sagernet/sing-box/option"
)

const (
	// defaultDataShards is the group size a proxy that asks for redundancy gets.
	// A stream is what this default is for: the parity of a group covers a loss
	// the stream would otherwise wait a round trip to have retransmitted, and the
	// wider the group the less the redundancy costs — thirty-two members with two
	// parity shards is a sixteenth of the traffic, and it measured better on a
	// lossy path than ten members with two did, both in what a short transfer
	// waits and in what the redundancy takes from a full one.
	defaultDataShards   = 32
	defaultParityShards = 2
	maxParityShards     = 8
)

// parseFEC turns the redundancy of the configuration into what the transport takes.
//
// The counts are checked here rather than in the transport: a group of one packet
// is a configuration that protects nothing, and more redundancy than members is a
// group sent more times than it has packets. The transport would take either.
func parseFEC(options option.QUICXFECOptions) (udpx.FECOptions, error) {
	if !options.Enabled {
		if options.DataShards != 0 || options.ParityShards != 0 {
			return udpx.FECOptions{}, E.New("fec: data_shards and parity_shards require enabled")
		}
		return udpx.FECOptions{}, nil
	}
	dataShards := options.DataShards
	if dataShards == 0 {
		dataShards = defaultDataShards
	}
	if dataShards < 2 {
		return udpx.FECOptions{}, E.New("fec: data_shards of ", dataShards)
	}
	parityShards := options.ParityShards
	if parityShards == 0 {
		parityShards = defaultParityShards
	}
	if parityShards < 0 {
		return udpx.FECOptions{}, E.New("fec: parity_shards of ", parityShards)
	}
	if parityShards > dataShards {
		return udpx.FECOptions{}, E.New("fec: ", parityShards, " parity shards for ", dataShards, " data shards")
	}
	if parityShards > maxParityShards {
		return udpx.FECOptions{}, E.New("fec: ", parityShards, " parity shards, at most ", maxParityShards)
	}
	return udpx.FECOptions{
		DataShards:   dataShards,
		ParityShards: parityShards,
	}, nil
}
