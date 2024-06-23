package iface

import (
	"errors"
	"fmt"
	"syscall"

	"golang.org/x/net/route"
)

// TODO: Deprecated
func DefaultRouteInterface() (string, error) {
	// or return ExeShell("route -n get default | grep 'interface' | awk 'NR==1{print $2}'")

	rib, err := route.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_DUMP2, 0)
	if err != nil {
		return "", fmt.Errorf("route.FetchRIB: %w", err)
	}
	msgs, err := route.ParseRIB(syscall.NET_RT_IFLIST2, rib)
	if err != nil {
		return "", fmt.Errorf("route.ParseRIB: %w", err)
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
		// default route
		if destination.IP != [4]byte{0, 0, 0, 0} {
			continue
		}
		// interface index
		if _, ok := addresses[1].(*route.Inet4Addr); ok {
			// ok
			if iface, err := GetInterfaceByIndex(routeMessage.Index); err == nil {
				return iface.Name, nil
			}
		}
	}

	return "", errors.New("ambiguous gateway interfaces found")
}
