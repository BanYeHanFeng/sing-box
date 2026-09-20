package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sagernet/quic-go"
	"github.com/sagernet/quic-go/qlog"
	"github.com/sagernet/quic-go/qlogwriter"
	boxTLS "github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/option"
	qtls "github.com/sagernet/sing-quic"
	"github.com/sagernet/sing-quic/quicx"
	"github.com/sagernet/sing/common/logger"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"

	"github.com/stretchr/testify/require"
)

// TestQUICXZeroRTT proves that a QUICX client really sends 0-RTT data. The box
// configuration has no switch for this, so the QUICX client and server are
// driven directly here with a qlog tracer injected, which is the only way to
// observe the packet types of a connection.
func TestQUICXZeroRTT(t *testing.T) {
	_, certPem, keyPem := createSelfSignedCertificate(t, "example.org")
	qlogDir := t.TempDir()
	t.Setenv("QLOGDIR", qlogDir)

	ctx := context.Background()
	serverTLS, err := boxTLS.NewServer(ctx, logger.NOP(), option.InboundTLSOptions{
		Enabled:         true,
		CertificatePath: certPem,
		KeyPath:         keyPem,
		ALPN:            []string{"h3"},
	})
	require.NoError(t, err)
	service, err := quicx.NewService[int](quicx.ServiceOptions{
		Context:     ctx,
		Logger:      logger.NOP(),
		TLSConfig:   serverTLS,
		QUICOptions: qtls.QUICOptions{},
		Handler:     quicxTestHandler{},
		Tracer:      quicxTestTracer,
	})
	require.NoError(t, err)
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	require.NoError(t, service.Start(packetConn))
	t.Cleanup(func() {
		service.Close()
		packetConn.Close()
	})

	clientTLS, err := boxTLS.NewClient(ctx, logger.NOP(), "example.org", option.OutboundTLSOptions{
		Enabled:    true,
		ServerName: "example.org",
		Insecure:   true,
		ALPN:       []string{"h3"},
	})
	require.NoError(t, err)
	client, err := quicx.NewClient(quicx.ClientOptions{
		Context:       ctx,
		Dialer:        &quicxTestDialer{},
		ServerAddress: M.SocksaddrFromNet(packetConn.LocalAddr()),
		TLSConfig:     clientTLS,
		QUICOptions:   qtls.QUICOptions{},
		Password:      "password",
		Tracer:        quicxTestTracer,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		client.CloseWithError(os.ErrClosed)
	})

	// A session ticket is required before 0-RTT can be attempted, and the client
	// has to install the cache for it on its own.
	stdConfig, err := clientTLS.STDConfig()
	require.NoError(t, err)
	require.NotNil(t, stdConfig.ClientSessionCache, "the QUICX client did not install a TLS session cache")
	cache := &quicxTestSessionCache{inner: stdConfig.ClientSessionCache}
	stdConfig.ClientSessionCache = cache

	var firstStream net.Conn
	for dial := 1; dial <= 2; dial++ {
		stream, err := client.DialConn(ctx, M.ParseSocksaddrHostPort("example.com", testPort))
		require.NoError(t, err)
		_, err = stream.Write([]byte("ping"))
		require.NoError(t, err)
		// The session ticket of the handshake arrives with a delay.
		time.Sleep(2 * time.Second)
		if dial == 1 {
			firstStream = stream
		} else {
			stream.Close()
		}
	}
	firstStream.Close()
	time.Sleep(500 * time.Millisecond)

	require.GreaterOrEqual(t, cache.gets, 2, "the second connection did not look for a resumed session")
	traces, err := filepath.Glob(filepath.Join(qlogDir, "*_client.sqlog"))
	require.NoError(t, err)
	require.NotEmpty(t, traces, "no client qlog traces were written")
	trace, err := os.ReadFile(quicxTestNewest(t, traces))
	require.NoError(t, err)
	require.Contains(t, string(trace), `"packet_type":"0RTT"`, "the second connection did not send any 0-RTT packet")
}

func quicxTestTracer(ctx context.Context, isClient bool, connID quic.ConnectionID) qlogwriter.Trace {
	return qlog.DefaultConnectionTracer(ctx, isClient, connID)
}

func quicxTestNewest(t *testing.T, paths []string) string {
	t.Helper()
	var newest string
	var newestTime time.Time
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newest, newestTime = path, info.ModTime()
		}
	}
	require.NotEmpty(t, newest, "no qlog trace found")
	return newest
}

// quicxTestSessionCache counts how often the installed cache is consulted.
type quicxTestSessionCache struct {
	inner  tls.ClientSessionCache
	access sync.Mutex
	gets   int
}

func (c *quicxTestSessionCache) Get(key string) (*tls.ClientSessionState, bool) {
	session, loaded := c.inner.Get(key)
	c.access.Lock()
	c.gets++
	c.access.Unlock()
	return session, loaded
}

func (c *quicxTestSessionCache) Put(key string, session *tls.ClientSessionState) {
	c.inner.Put(key, session)
}

type quicxTestDialer struct{}

func (d *quicxTestDialer) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "udp", destination.String())
}

func (d *quicxTestDialer) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return net.ListenPacket("udp", "")
}

type quicxTestHandler struct{}

func (quicxTestHandler) NewConnectionEx(ctx context.Context, conn net.Conn, source M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	go func() {
		defer onClose(nil)
		_, _ = io.Copy(io.Discard, conn)
	}()
}

func (quicxTestHandler) NewPacketConnectionEx(ctx context.Context, conn N.PacketConn, source M.Socksaddr, destination M.Socksaddr, onClose N.CloseHandlerFunc) {
	go func() {
		defer onClose(nil)
		<-ctx.Done()
	}()
}
