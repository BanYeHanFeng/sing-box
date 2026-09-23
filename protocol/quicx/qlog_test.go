package quicx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/qlog"
	"github.com/sagernet/sing/common/logger"

	"github.com/stretchr/testify/require"
)

// TestQLOGTracerTrace checks that the tracer writes a readable qlog trace for
// both perspectives, named after the connection id like quic-go does.
func TestQLOGTracerTrace(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "qlog")
	tracer, err := newQLOGTracer(logger.NOP(), directory)
	require.NoError(t, err)
	connID := quic.ConnectionIDFromBytes([]byte{0x01, 0x02, 0x03, 0x04})
	for _, testCase := range []struct {
		isClient    bool
		perspective string
	}{
		{isClient: true, perspective: "client"},
		{isClient: false, perspective: "server"},
	} {
		trace := tracer(context.Background(), testCase.isClient, connID)
		require.NotNil(t, trace, testCase.perspective)
		producer := trace.AddProducer()
		require.NotNil(t, producer, testCase.perspective)
		// Closing the last producer closes the trace and flushes the file.
		require.NoError(t, producer.Close(), testCase.perspective)

		content, err := os.ReadFile(filepath.Join(directory, connID.String()+"_"+testCase.perspective+".sqlog"))
		require.NoError(t, err, testCase.perspective)
		require.True(t, strings.HasPrefix(string(content), "\x1e"), testCase.perspective)
		require.Contains(t, string(content), `"file_schema":"urn:ietf:params:qlog:file:sequential"`, testCase.perspective)
		require.Contains(t, string(content), `"vantage_point":{"type":"`+testCase.perspective+`"}`, testCase.perspective)
		require.Contains(t, string(content), `"group_id":"`+connID.String()+`"`, testCase.perspective)
		require.Contains(t, string(content), `"event_schemas":["`+qlog.EventSchema+`"]`, testCase.perspective)
	}
}

// TestQLOGTracerDirectoryError checks that an unusable directory fails when the
// tracer is created, instead of leaving the instance running without traces.
func TestQLOGTracerDirectoryError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(path, []byte("qlog"), 0o644))
	_, err := newQLOGTracer(logger.NOP(), filepath.Join(path, "qlog"))
	require.Error(t, err)
}

// TestQLOGTracerFileError checks that a trace that cannot be created leaves the
// connection running untraced.
func TestQLOGTracerFileError(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "qlog")
	tracer, err := newQLOGTracer(logger.NOP(), directory)
	require.NoError(t, err)
	require.NoError(t, os.RemoveAll(directory))
	require.Nil(t, tracer(context.Background(), false, quic.ConnectionIDFromBytes([]byte{0x01})))
}
