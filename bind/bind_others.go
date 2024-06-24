//go:build !darwin && !linux && !windows

package bind

import (
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/josexy/cropstun/iface"
)

func lookupLocalAddr(ifaceName string, network string, dst netip.Addr, port uint16) (netip.AddrPort, error) {
	iface, err := iface.GetInterfaceByName(ifaceName)
	if err != nil {
		return netip.AddrPort{}, err
	}
	var addr netip.Addr
	switch network {
	case "udp4", "tcp4":
		addr = iface.PickIPv4Addr(dst)
	case "tcp6", "udp6":
		addr = iface.PickIPv6Addr(dst)
	default:
		if dst.IsValid() {
			if dst.Is4() {
				addr = iface.PickIPv4Addr(dst)
			} else {
				addr = iface.PickIPv6Addr(dst)
			}
		} else {
			addr = iface.PickIPv4Addr(dst)
		}
	}
	if !addr.IsValid() {
		return netip.AddrPort{}, ErrInvalidIPAddr
	}
	return netip.AddrPortFrom(addr, uint16(port)), nil
}

func bindToDeviceForConn(ifaceName string, dialer *net.Dialer, network string, dst netip.Addr) error {
	if !dst.IsGlobalUnicast() {
		return nil
	}

	// local dynamic port
	localPort := uint16(0)

	if dialer.LocalAddr != nil {
		if _, port, err := net.SplitHostPort(dialer.LocalAddr.String()); err == nil {
			port, _ := strconv.ParseUint(port, 10, 16)
			localPort = uint16(port)
		}
	}

	addrPort, err := lookupLocalAddr(ifaceName, network, dst, localPort)
	if err != nil {
		return err
	}

	var addr net.Addr
	if strings.HasPrefix(network, "tcp") {
		addr = &net.TCPAddr{
			IP:   addrPort.Addr().AsSlice(),
			Port: int(addrPort.Port()),
		}
	} else if strings.HasPrefix(network, "udp") {
		addr = &net.UDPAddr{
			IP:   addrPort.Addr().AsSlice(),
			Port: int(addrPort.Port()),
		}
	}

	dialer.LocalAddr = addr
	return nil
}

func bindToDeviceForPacket(ifaceName string, _ *net.ListenConfig, network, address string) (string, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		port = "0"
	}

	localPort, _ := strconv.ParseUint(port, 10, 16)
	addr, err := lookupLocalAddr(ifaceName, network, netip.Addr{}, uint16(localPort))
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}
