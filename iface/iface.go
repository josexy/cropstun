package iface

import (
	"fmt"
	"net"
	"net/netip"
	"sync"
)

var (
	mu     sync.RWMutex
	record map[string]*Interface
)

type Interface struct {
	Index        int
	Name         string
	MTU          int
	Addrs        []netip.Prefix
	Addrsv4      []netip.Prefix
	Addrsv6      []netip.Prefix
	HardwareAddr net.HardwareAddr
}

func init() {
	_ = FlushAllInterfaces()
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
	mu.RLock()
	defer mu.RUnlock()
	for _, iface := range record {
		if iface.Index == index {
			return iface, nil
		}
	}
	return nil, fmt.Errorf("interface index %d not found", index)
}

func GetInterfaceByName(name string) (*Interface, error) {
	mu.RLock()
	defer mu.RUnlock()
	if iface, ok := record[name]; ok {
		return iface, nil
	}
	return nil, fmt.Errorf("interface name %q not found", name)
}

func GetAllInterfaceNames() (list []string) {
	mu.RLock()
	defer mu.RUnlock()
	for k := range record {
		list = append(list, k)
	}
	return
}

func GetAllInterfaces() (list []*Interface) {
	mu.RLock()
	defer mu.RUnlock()
	for _, v := range record {
		niface := *v
		niface.HardwareAddr = append([]byte(nil), niface.HardwareAddr...)
		list = append(list, &niface)
	}
	return
}

func FlushAllInterfaces() error {
	mu.Lock()
	defer mu.Unlock()
	if record == nil {
		record = make(map[string]*Interface)
	}
	clear(record)

	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		var cidrsv4, cidrsv6 []netip.Prefix
		for _, addr := range addrs {
			prefix, err := netip.ParsePrefix(addr.String())
			if err != nil {
				continue
			}
			if prefix.Addr().Is4() {
				cidrsv4 = append(cidrsv4, prefix)
			} else {
				cidrsv6 = append(cidrsv6, prefix)
			}
		}
		allcidrs := make([]netip.Prefix, 0, len(cidrsv4)+len(cidrsv6))
		allcidrs = append(allcidrs, cidrsv4...)
		allcidrs = append(allcidrs, cidrsv6...)
		record[iface.Name] = &Interface{
			Index:        iface.Index,
			Name:         iface.Name,
			MTU:          iface.MTU,
			Addrs:        allcidrs,
			Addrsv4:      cidrsv4,
			Addrsv6:      cidrsv6,
			HardwareAddr: iface.HardwareAddr,
		}
	}
	return nil
}
