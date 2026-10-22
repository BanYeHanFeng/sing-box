package quicx

import (
	"time"

	"github.com/sagernet/sing-box/option"
	quicxlib "github.com/sagernet/sing-quic/quicx"
)

func buildFECOptions(options option.QUICXFECOptions) quicxlib.FECOptions {
	return quicxlib.FECOptions{
		Enabled:      options.Enabled,
		Block:        options.Block,
		BlockSize:    options.BlockSize,
		Parity:       options.Parity,
		MaxParity:    options.MaxParity,
		Interleave:   options.Interleave,
		BlockTimeout: time.Duration(options.BlockTimeout),
	}
}
