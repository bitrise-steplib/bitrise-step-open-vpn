package openvpn

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderClientConfig(t *testing.T) {
	conn := Connection{
		Host:  "vpn.example.com",
		Port:  1194,
		Proto: "udp",
	}

	got := RenderClientConfig(conn)

	want := `client
dev tun
proto udp
remote vpn.example.com 1194
resolv-retry infinite
nobind
persist-key
persist-tun
comp-lzo
verb 3
ca ca.crt
cert client.crt
key client.key
`
	assert.Equal(t, want, got)
}

func TestRenderClientConfig_proto_and_port(t *testing.T) {
	conn := Connection{Host: "10.0.0.1", Port: 443, Proto: "tcp"}

	got := RenderClientConfig(conn)

	assert.Contains(t, got, "proto tcp")
	assert.Contains(t, got, "remote 10.0.0.1 443")
}

func TestDarwinArgs(t *testing.T) {
	conn := Connection{Host: "vpn.example.com", Port: 1194, Proto: "udp"}

	got := darwinArgs(conn)

	want := []string{
		"openvpn",
		"--client",
		"--dev", "tun",
		"--proto", "udp",
		"--remote", "vpn.example.com", "1194",
		"--resolv-retry", "infinite",
		"--nobind",
		"--persist-key",
		"--persist-tun",
		"--comp-lzo",
		"--verb", "3",
		"--ca", "ca.crt",
		"--cert", "client.crt",
		"--key", "client.key",
	}
	require.Equal(t, want, got)
	// openvpn must be the first arg so it is the command sudo executes.
	assert.Equal(t, "openvpn", got[0])
	// The port is rendered right after the host in --remote.
	remoteIdx := indexOf(got, "--remote")
	require.GreaterOrEqual(t, remoteIdx, 0)
	assert.Equal(t, "vpn.example.com", got[remoteIdx+1])
	assert.Equal(t, "1194", got[remoteIdx+2])
}

func TestConnect_unsupportedOS(t *testing.T) {
	connector := NewConnector("windows", nil, nil, nil)

	err := connector.Connect(Connection{}, "")

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "unsupported operating system"))
}

func indexOf(s []string, v string) int {
	for i, e := range s {
		if e == v {
			return i
		}
	}
	return -1
}
