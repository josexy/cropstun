package route

import (
	"net"
	"syscall"

	"golang.org/x/net/route"
)

func DefaultRouteInterface() (*Route, error) {
	rib, err := route.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_DUMP2, 0)
	if err != nil {
		return nil, err
	}
	msgs, err := route.ParseRIB(syscall.NET_RT_IFLIST2, rib)
	if err != nil {
		return nil, err
	}
	for _, message := range msgs {
		routeMessage := message.(*route.RouteMessage)
		if routeMessage.Flags&(syscall.RTF_UP|syscall.RTF_GATEWAY|syscall.RTF_STATIC) == 0 {
			continue
		}
		addresses := routeMessage.Addrs
		destination, ok := addresses[0].(*route.Inet4Addr)
		if !ok {
			continue
		}
		if destination.IP != [4]byte{0, 0, 0, 0} {
			continue
		}
		if _, ok := addresses[1].(*route.Inet4Addr); ok {
			if iface, err := net.InterfaceByIndex(routeMessage.Index); err == nil {
				return &Route{
					InterfaceName:  iface.Name,
					InterfaceIndex: iface.Index,
				}, nil
			} else {
				return nil, err
			}
		}
	}

	return nil, ErrNoRoute
}
