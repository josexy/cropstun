package bind

import (
	"context"
	"net"
	"net/netip"
	"syscall"

	"golang.org/x/sys/unix"
)

func setupControl(ifaceName string) controlFn {
	return func(ctx context.Context, network, address string, c syscall.RawConn) (err error) {
		addrPort, err := netip.ParseAddrPort(address)
		if err == nil && !addrPort.Addr().IsGlobalUnicast() {
			return
		}

		var innerErr error
		err = c.Control(func(fd uintptr) {
			innerErr = unix.BindToDevice(int(fd), ifaceName)
		})
		if innerErr != nil {
			err = innerErr
		}
		return
	}
}

func bindToDeviceForConn(ifaceName string, dialer *net.Dialer, _ string, _ netip.Addr) error {
	addControlToDialer(dialer, setupControl(ifaceName))
	return nil
}

func bindToDeviceForPacket(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	addControlToListenConfig(lc, setupControl(ifaceName))
	return address, nil
}
