package tun

import (
	"net/netip"
	"runtime"
)

func (o *Options) BuildAutoRouteRanges() ([]netip.Prefix, error) {
	var routeRanges []netip.Prefix
	if len(o.Inet4Address) > 0 {
		var inet4Ranges []netip.Prefix
		if runtime.GOOS == "darwin" {
			inet4Ranges = []netip.Prefix{
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 1}), 8),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 2}), 7),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 4}), 6),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 8}), 5),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 16}), 4),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 32}), 3),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 64}), 2),
				netip.PrefixFrom(netip.AddrFrom4([4]byte{0: 128}), 1),
			}
		} else {
			inet4Ranges = []netip.Prefix{netip.PrefixFrom(netip.IPv4Unspecified(), 0)}
		}
		routeRanges = append(routeRanges, inet4Ranges...)
	}
	if len(o.Inet6Address) > 0 {
		var inet6Ranges []netip.Prefix
		if runtime.GOOS == "darwin" {
			inet6Ranges = []netip.Prefix{
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 1}), 8),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 2}), 7),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 4}), 6),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 8}), 5),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 16}), 4),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 32}), 3),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 64}), 2),
				netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 128}), 1),
			}
		} else {
			inet6Ranges = []netip.Prefix{netip.PrefixFrom(netip.IPv6Unspecified(), 0)}
		}
		routeRanges = append(routeRanges, inet6Ranges...)
	}
	return routeRanges, nil
}
