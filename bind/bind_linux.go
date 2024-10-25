package bind

import (
	"context"
	"net"
	"net/netip"
	"syscall"

	"github.com/josexy/cropstun/iface"
)

func setupControl(ifaceName string) controlFn {
	return func(ctx context.Context, network, address string, c syscall.RawConn) (err error) {
		addrPort, err := netip.ParseAddrPort(address)
		if err == nil && !addrPort.Addr().IsGlobalUnicast() {
			return
		}

		var innerErr error
		err = c.Control(func(fd uintptr) {
			innerErr = syscall.BindToDevice(int(fd), ifaceName)
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
	addControlToDialer(dialer, setupControl(iface.Name))
	return nil
}

func bindToDeviceForPacket(ifaceName string, lc *net.ListenConfig) error {
	iface, err := iface.GetInterfaceByName(ifaceName)
	if err != nil {
		return err
	}
	addControlToListenConfig(lc, setupControl(iface.Name))
	return nil
}
