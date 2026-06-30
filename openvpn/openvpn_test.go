package openvpn

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenvpnArgs(t *testing.T) {
	conn := Connection{Host: "vpn.example.com", Port: 1194, Proto: "udp"}

	got := openvpnArgs(conn, "/tmp/certs/ca.crt", "/tmp/certs/client.crt", "/tmp/certs/client.key")

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
		"--ca", "/tmp/certs/ca.crt",
		"--cert", "/tmp/certs/client.crt",
		"--key", "/tmp/certs/client.key",
	}
	require.Equal(t, want, got)
	// openvpn must be the first arg so it is the command sudo executes.
	assert.Equal(t, "openvpn", got[0])
}

func TestOpenvpnArgs_protoAndPort(t *testing.T) {
	conn := Connection{Host: "10.0.0.1", Port: 443, Proto: "tcp"}

	got := openvpnArgs(conn, "ca", "cert", "key")

	remoteIdx := indexOf(got, "--remote")
	require.GreaterOrEqual(t, remoteIdx, 0)
	assert.Equal(t, "10.0.0.1", got[remoteIdx+1])
	assert.Equal(t, "443", got[remoteIdx+2])

	protoIdx := indexOf(got, "--proto")
	require.GreaterOrEqual(t, protoIdx, 0)
	assert.Equal(t, "tcp", got[protoIdx+1])
}

// openvpnArgs must never reference /etc/openvpn: keys live in a writable
// temporary directory so the Step works without root.
func TestOpenvpnArgs_noPrivilegedPaths(t *testing.T) {
	conn := Connection{Host: "vpn.example.com", Port: 1194, Proto: "udp"}

	got := openvpnArgs(conn, "/tmp/certs/ca.crt", "/tmp/certs/client.crt", "/tmp/certs/client.key")

	for _, arg := range got {
		assert.NotContains(t, arg, "/etc/openvpn")
	}
}

func TestNewConnector_unsupportedOS(t *testing.T) {
	connector, err := NewConnector("windows", nil, nil, nil, nil)

	require.Error(t, err)
	assert.Nil(t, connector)
	assert.Contains(t, err.Error(), "unsupported operating system")
}

func indexOf(s []string, v string) int {
	for i, e := range s {
		if e == v {
			return i
		}
	}
	return -1
}
