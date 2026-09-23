//go:build with_quic

package main

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/json/badoption"

	"github.com/stretchr/testify/require"
)

// TestQUICXSelf runs the QUICX inbound and outbound of one instance against
// each other, including a client restart that forces a new QUIC connection. The
// suite covers TCP, UDP and UDP messages which have to be fragmented.
func TestQUICXSelf(t *testing.T) {
	caPem, certPem, keyPem := createSelfSignedCertificate(t, "example.org")
	caPemContent, err := os.ReadFile(caPem)
	require.NoError(t, err)
	startInstance(t, option.Options{
		Inbounds: []option.Inbound{
			{
				Type: C.TypeMixed,
				Tag:  "mixed-in",
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.IPv4Unspecified())),
						ListenPort: clientPort,
					},
				},
			},
			{
				Type: C.TypeQUICX,
				Tag:  "quicx-in",
				Options: &option.QUICXInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.IPv4Unspecified())),
						ListenPort: serverPort,
					},
					Users: []option.QUICXUser{
						{
							Name:     "sekai",
							Password: "password",
						},
					},
					InboundTLSOptionsContainer: option.InboundTLSOptionsContainer{
						TLS: &option.InboundTLSOptions{
							Enabled:         true,
							ServerName:      "example.org",
							CertificatePath: certPem,
							KeyPath:         keyPem,
						},
					},
				},
			},
		},
		Outbounds: []option.Outbound{
			{
				Type: C.TypeDirect,
			},
			{
				Type: C.TypeQUICX,
				Tag:  "quicx-out",
				Options: &option.QUICXOutboundOptions{
					ServerOptions: option.ServerOptions{
						Server:     "127.0.0.1",
						ServerPort: serverPort,
					},
					Password: "password",
					OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{
						TLS: &option.OutboundTLSOptions{
							Enabled:     true,
							ServerName:  "example.org",
							Certificate: []string{string(caPemContent)},
						},
					},
				},
			},
		},
		Route: &option.RouteOptions{
			Rules: []option.Rule{
				{
					Type: C.RuleTypeDefault,
					DefaultOptions: option.DefaultRule{
						RawDefaultRule: option.RawDefaultRule{
							Inbound: []string{"mixed-in"},
						},
						RuleAction: option.RuleAction{
							Action: C.RuleActionTypeRoute,
							RouteOptions: option.RouteActionOptions{
								Outbound: "quicx-out",
							},
						},
					},
				},
			},
		},
	})
	testSuit(t, clientPort, testPort)
}

// TestQUICXQLOG turns on qlog_directory on both the QUICX inbound and outbound
// and checks that both sides write a standard qlog trace for the connection the
// proxy traffic uses.
func TestQUICXQLOG(t *testing.T) {
	caPem, certPem, keyPem := createSelfSignedCertificate(t, "example.org")
	caPemContent, err := os.ReadFile(caPem)
	require.NoError(t, err)
	serverQLOGDirectory := t.TempDir()
	clientQLOGDirectory := t.TempDir()
	instance := startInstance(t, option.Options{
		Inbounds: []option.Inbound{
			{
				Type: C.TypeMixed,
				Tag:  "mixed-in",
				Options: &option.HTTPMixedInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.IPv4Unspecified())),
						ListenPort: clientPort,
					},
				},
			},
			{
				Type: C.TypeQUICX,
				Tag:  "quicx-in",
				Options: &option.QUICXInboundOptions{
					ListenOptions: option.ListenOptions{
						Listen:     common.Ptr(badoption.Addr(netip.IPv4Unspecified())),
						ListenPort: serverPort,
					},
					Users: []option.QUICXUser{
						{
							Name:     "sekai",
							Password: "password",
						},
					},
					QLOGDirectory: serverQLOGDirectory,
					InboundTLSOptionsContainer: option.InboundTLSOptionsContainer{
						TLS: &option.InboundTLSOptions{
							Enabled:         true,
							ServerName:      "example.org",
							CertificatePath: certPem,
							KeyPath:         keyPem,
						},
					},
				},
			},
		},
		Outbounds: []option.Outbound{
			{
				Type: C.TypeDirect,
			},
			{
				Type: C.TypeQUICX,
				Tag:  "quicx-out",
				Options: &option.QUICXOutboundOptions{
					ServerOptions: option.ServerOptions{
						Server:     "127.0.0.1",
						ServerPort: serverPort,
					},
					Password:      "password",
					QLOGDirectory: clientQLOGDirectory,
					OutboundTLSOptionsContainer: option.OutboundTLSOptionsContainer{
						TLS: &option.OutboundTLSOptions{
							Enabled:     true,
							ServerName:  "example.org",
							Certificate: []string{string(caPemContent)},
						},
					},
				},
			},
		},
		Route: &option.RouteOptions{
			Rules: []option.Rule{
				{
					Type: C.RuleTypeDefault,
					DefaultOptions: option.DefaultRule{
						RawDefaultRule: option.RawDefaultRule{
							Inbound: []string{"mixed-in"},
						},
						RuleAction: option.RuleAction{
							Action: C.RuleActionTypeRoute,
							RouteOptions: option.RouteActionOptions{
								Outbound: "quicx-out",
							},
						},
					},
				},
			},
		},
	})
	testSuitSimple(t, clientPort, testPort)
	// A trace is only flushed when its connection closes, which happens with
	// the instance here; the cleanup closes the already closed instance again.
	_ = instance.Close()
	quicxTestWaitForTrace(t, clientQLOGDirectory, "client")
	quicxTestWaitForTrace(t, serverQLOGDirectory, "server")
}

// quicxTestWaitForTrace waits until directory holds a complete qlog trace of
// the given perspective. A trace is complete once its header and at least one
// event record are on disk, which only happens after its connection closed.
func quicxTestWaitForTrace(t *testing.T, directory string, perspective string) {
	t.Helper()
	require.Eventually(t, func() bool {
		paths, err := filepath.Glob(filepath.Join(directory, "*_"+perspective+".sqlog"))
		if err != nil {
			return false
		}
		for _, path := range paths {
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if strings.Contains(string(content), `"vantage_point":{"type":"`+perspective+`"}`) &&
				strings.Count(string(content), "\x1e") > 1 {
				return true
			}
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "no complete %s qlog trace was written to %s", perspective, directory)
}
