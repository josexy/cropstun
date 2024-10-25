package route

import (
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func DefaultRouteInterface() (*Route, error) {
	routes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL,
		&netlink.Route{Table: unix.RT_TABLE_MAIN}, netlink.RT_FILTER_TABLE)
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		if route.Dst == nil {
			continue
		}
		if route.Family == netlink.FAMILY_V4 && !route.Dst.IP.Equal(net.IPv4zero) {
			continue
		} else if route.Family == netlink.FAMILY_V6 && !route.Dst.IP.Equal(net.IPv6zero) {
			continue
		}

		var link netlink.Link
		link, err = netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			return nil, err
		}

		return &Route{
			InterfaceName:  link.Attrs().Name,
			InterfaceIndex: link.Attrs().Index,
		}, nil
	}
	return nil, ErrNoRoute
}
