package bind

import (
	"net"
	"net/netip"
	"syscall"

	"github.com/josexy/cropstun/iface"
	"golang.org/x/sys/unix"
)

type controlFn func(network, address string, c syscall.RawConn) error

func setupControl(ifaceIndex int, nextChain controlFn) controlFn {
	return func(network, address string, c syscall.RawConn) (err error) {
		defer func() {
			if err == nil && nextChain != nil {
				err = nextChain(network, address, c)
			}
		}()

		ipStr, _, err := net.SplitHostPort(address)
		if err == nil {
			if ip, err := netip.ParseAddr(ipStr); err == nil && !ip.IsGlobalUnicast() {
				return err
			}
		}

		// there is no SO_BINDTODEVICE on macOS
		var innerErr error
		err = c.Control(func(fd uintptr) {
			switch network {
			case "tcp4", "udp4":
				innerErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, ifaceIndex)
			case "tcp6", "udp6":
				innerErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, ifaceIndex)
			}
		})
		if innerErr != nil {
			err = innerErr
		}
		return
	}
}

func bindToDeviceForConn(ifaceName string, dialer *net.Dialer, _ string, _ netip.Addr) error {
	iface, err := iface.GetInterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	dialer.Control = setupControl(iface.Index, dialer.Control)
	return nil
}

func bindToDeviceForPacket(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	iface, err := iface.GetInterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}
	lc.Control = setupControl(iface.Index, lc.Control)
	return address, nil
}
