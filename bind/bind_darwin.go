package bind

import (
	"context"
	"net"
	"net/netip"
	"syscall"

	"github.com/josexy/cropstun/iface"
)

func setupControl(ifaceIndex int) controlFn {
	return func(ctx context.Context, network, address string, c syscall.RawConn) (err error) {
		addrPort, err := netip.ParseAddrPort(address)
		if err == nil && !addrPort.Addr().IsGlobalUnicast() {
			return
		}

		var innerErr error
		err = c.Control(func(fd uintptr) {
			switch network {
			case "tcp4", "udp4":
				innerErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_BOUND_IF, ifaceIndex)
			case "tcp6", "udp6":
				innerErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_BOUND_IF, ifaceIndex)
			}
		})
		if innerErr != nil {
			err = innerErr
		}
		return
	}
}

func bindToDeviceForConn(ifaceName string, dialer *net.Dialer) error {
	iface, err := iface.GetInterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	addControlToDialer(dialer, setupControl(iface.Index))
	return nil
}

func bindToDeviceForPacket(ifaceName string, lc *net.ListenConfig) error {
	iface, err := iface.GetInterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	addControlToListenConfig(lc, setupControl(iface.Index))
	return nil
}
