package iface

import (
	"fmt"
	"net"
	"net/netip"
	"sync"
)

var (
	once    sync.Once
	onceErr error
	record  map[string]*Interface
)

type Interface struct {
	Index        int
	Name         string
	Addrs        []netip.Prefix
	HardwareAddr net.HardwareAddr
	hasIPv4Addr  bool
}

func init() {
	resolveAllInterfaces()
	if onceErr != nil {
		panic(onceErr)
	}
}

func (iface *Interface) PickIPv4Addr(dst netip.Addr) netip.Addr {
	return iface.pickIPAddr(dst, func(addr netip.Prefix) bool {
		return addr.Addr().Is4()
	})
}

func (iface *Interface) PickIPv6Addr(dst netip.Addr) netip.Addr {
	return iface.pickIPAddr(dst, func(addr netip.Prefix) bool {
		return addr.Addr().Is6()
	})
}

// pickIPAddr pick a valid ip address from all interfaces which serve as outbound address
func (iface *Interface) pickIPAddr(dst netip.Addr, accept func(addr netip.Prefix) bool) netip.Addr {
	var fallback netip.Addr

	for _, addr := range iface.Addrs {
		if !accept(addr) {
			continue
		}

		// ignore link-local unicast address
		// ipv4: 169.254.0.0/16
		// ipv6: fe80::/10
		if !fallback.IsValid() && !addr.Addr().IsLinkLocalUnicast() {
			fallback = addr.Addr()
			// this case is occur when pick a UDP local address
			if !dst.IsValid() {
				break
			}
		}

		// the destination and picked address at same subnet
		// so it is easy to return the trigged interface address
		if dst.IsValid() && addr.Contains(dst) {
			return addr.Addr()
		}
	}

	return fallback
}

func GetInterfaceByIndex(index int) (*Interface, error) {
	for _, iface := range record {
		if iface.Index == index {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface index %d not found", index)
}

func GetInterfaceByName(name string) (*Interface, error) {
	if iface, ok := record[name]; ok {
		return iface, nil
	}
	return nil, fmt.Errorf("interface name %q not found", name)
}

func GetInterfaceNames() (list []string) {
	for k, v := range record {
		if v.hasIPv4Addr {
			list = append(list, k)
		}
	}
	return
}

func resolveAllInterfaces() {
	once.Do(func() {
		record = make(map[string]*Interface)
		ifaces, err := net.Interfaces()
		if err != nil {
			onceErr = err
			return
		}
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			onceErr = err
			if err != nil || len(addrs) == 0 {
				continue
			}
			ipNets := make([]netip.Prefix, 0, len(addrs))
			hasIPv4Addr := false
			for _, addr := range addrs {
				ipNet := netip.MustParsePrefix(addr.String())
				if ipNet.Addr().Is4() {
					hasIPv4Addr = true
				}
				ipNets = append(ipNets, ipNet)
			}
			record[iface.Name] = &Interface{
				Index:        iface.Index,
				Name:         iface.Name,
				Addrs:        ipNets,
				HardwareAddr: iface.HardwareAddr,
				hasIPv4Addr:  hasIPv4Addr,
			}
		}
	})
}
