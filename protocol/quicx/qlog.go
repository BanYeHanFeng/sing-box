package quicx

import (
	"bufio"
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/qlog"
	"github.com/sagernet/quic-go/qlogwriter"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/logger"
)

// qlogTracer is the per-connection tracer accepted by the QUICX client and
// service. It is called once for every QUIC connection and returns the trace
// the connection records its events into, or nil to leave it untraced.
type qlogTracer = func(ctx context.Context, isClient bool, connID quic.ConnectionID) qlogwriter.Trace

// newQLOGTracer returns a tracer writing one qlog trace per QUIC connection
// into directory, named after the original destination connection id like
// quic-go's QLOGDIR: <connection id>_<client|server>.sqlog.
//
// The directory is created here, so a path that cannot be used fails the
// configuration instead of silently disabling tracing. Every connection gets
// its own file and nothing prunes the directory, so it only grows: the option
// is meant for debugging a running instance, not for permanent operation.
func newQLOGTracer(instanceLogger logger.Logger, directory string) (qlogTracer, error) {
	err := os.MkdirAll(directory, 0o755)
	if err != nil {
		return nil, E.Cause(err, "create qlog directory ", directory)
	}
	return func(_ context.Context, isClient bool, connID quic.ConnectionID) qlogwriter.Trace {
		perspective := "server"
		if isClient {
			perspective = "client"
		}
		path := filepath.Join(directory, connID.String()+"_"+perspective+".sqlog")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			instanceLogger.Error(E.Cause(err, "create qlog file ", path))
			return nil
		}
		trace := qlogwriter.NewConnectionFileSeq(newQLOGFile(file), isClient, connID, []string{qlog.EventSchema})
		go trace.Run()
		return trace
	}, nil
}

// qlogFile buffers the events of one trace: quic-go encodes the qlog JSON
// token by token, so without buffering every connection would issue one write
// syscall per token. quic-go closes the trace together with its connection,
// which flushes the buffer here.
type qlogFile struct {
	writer *bufio.Writer
	file   *os.File
}

func newQLOGFile(file *os.File) *qlogFile {
	return &qlogFile{
		writer: bufio.NewWriter(file),
		file:   file,
	}
}

func (f *qlogFile) Write(p []byte) (int, error) {
	return f.writer.Write(p)
}

func (f *qlogFile) Close() error {
	return errors.Join(f.writer.Flush(), f.file.Close())
}
