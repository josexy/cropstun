package bind

import (
	"net"
	"net/netip"
)

func BindToDeviceForConn(ifaceName string, dialer *net.Dialer, network string, dst netip.Addr) error {
	return bindToDeviceForConn(ifaceName, dialer, network, dst)
}

func BindToDeviceForPacket(ifaceName string, lc *net.ListenConfig, network, dst string) (string, error) {
	return bindToDeviceForPacket(ifaceName, lc, network, dst)
}
