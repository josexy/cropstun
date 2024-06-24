package bind

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"syscall"
)

var ErrInvalidIPAddr = errors.New("invalid ip address")

func BindToDeviceForConn(ifaceName string, dialer *net.Dialer, network string, dst netip.Addr) error {
	return bindToDeviceForConn(ifaceName, dialer, network, dst)
}

func BindToDeviceForPacket(ifaceName string, lc *net.ListenConfig, network, dst string) (string, error) {
	return bindToDeviceForPacket(ifaceName, lc, network, dst)
}

type controlFn = func(ctx context.Context, network, address string, c syscall.RawConn) error

func addControlToListenConfig(lc *net.ListenConfig, fn controlFn) {
	llc := *lc
	lc.Control = func(network, address string, c syscall.RawConn) (err error) {
		switch {
		case llc.Control != nil:
			if err = llc.Control(network, address, c); err != nil {
				return
			}
		}
		return fn(context.Background(), network, address, c)
	}
}

func addControlToDialer(d *net.Dialer, fn controlFn) {
	ld := *d
	d.ControlContext = func(ctx context.Context, network, address string, c syscall.RawConn) (err error) {
		switch {
		case ld.ControlContext != nil:
			if err = ld.ControlContext(ctx, network, address, c); err != nil {
				return
			}
		case ld.Control != nil:
			if err = ld.Control(network, address, c); err != nil {
				return
			}
		}
		return fn(ctx, network, address, c)
	}
}
