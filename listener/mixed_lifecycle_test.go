package listener

import (
	"net"
	"strconv"
	"testing"

	"github.com/metacubex/mihomo/tunnel"
)

func TestMixedPortIsNotReportedAfterUDPBindFailure(t *testing.T) {
	SetAllowLan(false)
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, rawPort, err := net.SplitHostPort(udp.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	defer ReCreateMixed(0, tunnel.Tunnel)
	ReCreateMixed(port, tunnel.Tunnel)
	if got := GetPorts().MixedPort; got != 0 {
		t.Fatalf("failed mixed listener reported port %d", got)
	}
	if err := udp.Close(); err != nil {
		t.Fatal(err)
	}
	ReCreateMixed(port, tunnel.Tunnel)
	if got := GetPorts().MixedPort; got != port {
		t.Fatalf("retry reported %d, want %d", got, port)
	}
}
