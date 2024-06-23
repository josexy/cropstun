package bind

import (
	"net"
	"net/netip"
	"syscall"

	"golang.org/x/sys/unix"
)

type controlFn func(network, address string, c syscall.RawConn) error

func setupControl(ifaceName string, nextChain controlFn) controlFn {
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
	dialer.Control = setupControl(ifaceName, dialer.Control)
	return nil
}

func bindToDeviceForPacket(ifaceName string, lc *net.ListenConfig, _, address string) (string, error) {
	lc.Control = setupControl(ifaceName, lc.Control)
	return address, nil
}
