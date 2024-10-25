package bind

import (
	"context"
	"errors"
	"net"
	"syscall"
)

var ErrInvalidIPAddr = errors.New("invalid ip address")

func BindToDeviceForConn(ifaceName string, dialer *net.Dialer) error {
	return bindToDeviceForConn(ifaceName, dialer)
}

func BindToDeviceForPacket(ifaceName string, lc *net.ListenConfig) error {
	return bindToDeviceForPacket(ifaceName, lc)
}

type controlFn = func(ctx context.Context, network, address string, c syscall.RawConn) error

func addControlToListenConfig(lc *net.ListenConfig, fn controlFn) {
	olc := *lc
	lc.Control = func(network, address string, c syscall.RawConn) (err error) {
		switch {
		case olc.Control != nil:
			if err = olc.Control(network, address, c); err != nil {
				return
			}
		}
		return fn(context.Background(), network, address, c)
	}
}

func addControlToDialer(d *net.Dialer, fn controlFn) {
	od := *d
	d.ControlContext = func(ctx context.Context, network, address string, c syscall.RawConn) (err error) {
		switch {
		case od.ControlContext != nil:
			if err = od.ControlContext(ctx, network, address, c); err != nil {
				return
			}
		case od.Control != nil:
			if err = od.Control(network, address, c); err != nil {
				return
			}
		}
		return fn(ctx, network, address, c)
	}
}
