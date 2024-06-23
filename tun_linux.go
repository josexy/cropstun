package tun

import (
	"math/rand"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"unsafe"

	"github.com/josexy/cropstun/common/buf"
	"github.com/josexy/cropstun/common/bufio"
	N "github.com/josexy/cropstun/common/network"
	"github.com/vishvananda/netlink"

	"golang.org/x/sys/unix"
)

var _ LinuxTUN = (*NativeTun)(nil)

type NativeTun struct {
	tunFd             int
	tunFile           *os.File
	tunWriter         N.VectorisedWriter
	options           *Options
	ruleIndex6        []int
	txChecksumOffload bool
}

func New(options *Options) (Tun, error) {
	var nativeTun *NativeTun
	tunFd, err := open(options.Name)
	if err != nil {
		return nil, err
	}
	tunLink, err := netlink.LinkByName(options.Name)
	if err != nil {
		return nil, err
	}
	nativeTun = &NativeTun{
		tunFd:   tunFd,
		tunFile: os.NewFile(uintptr(tunFd), "tun"),
		options: options,
	}
	err = nativeTun.configure(tunLink)
	if err != nil {
		return nil, err
	}
	var ok bool
	nativeTun.tunWriter, ok = bufio.CreateVectorisedWriter(nativeTun.tunFile)
	if !ok {
		panic("create vectorised writer")
	}
	return nativeTun, nil
}

func (t *NativeTun) Read(p []byte) (n int, err error) {
	return t.tunFile.Read(p)
}

func (t *NativeTun) Write(p []byte) (n int, err error) {
	return t.tunFile.Write(p)
}

func (t *NativeTun) WriteVectorised(buffers []*buf.Buffer) error {
	return t.tunWriter.WriteVectorised(buffers)
}

func open(name string) (int, error) {
	fd, err := unix.Open("/dev/net/tun", unix.O_RDWR, 0)
	if err != nil {
		return -1, err
	}

	var ifr struct {
		name  [16]byte
		flags uint16
		_     [22]byte
	}

	copy(ifr.name[:], name)
	ifr.flags = unix.IFF_TUN | unix.IFF_NO_PI
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), unix.TUNSETIFF, uintptr(unsafe.Pointer(&ifr)))
	if errno != 0 {
		unix.Close(fd)
		return -1, errno
	}

	if err = unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return -1, err
	}

	return fd, nil
}

func (t *NativeTun) configure(tunLink netlink.Link) error {
	err := netlink.LinkSetMTU(tunLink, int(t.options.MTU))
	if err == unix.EPERM {
		// unprivileged
		return nil
	} else if err != nil {
		return err
	}

	if len(t.options.Inet4Address) > 0 {
		for _, address := range t.options.Inet4Address {
			addr4, _ := netlink.ParseAddr(address.String())
			err = netlink.AddrAdd(tunLink, addr4)
			if err != nil {
				return err
			}
		}
	}
	if len(t.options.Inet6Address) > 0 {
		for _, address := range t.options.Inet6Address {
			addr6, _ := netlink.ParseAddr(address.String())
			err = netlink.AddrAdd(tunLink, addr6)
			if err != nil {
				return err
			}
		}
	}

	var rxChecksumOffload bool
	rxChecksumOffload, err = checkChecksumOffload(t.options.Name, unix.ETHTOOL_GRXCSUM)
	if err == nil && !rxChecksumOffload {
		_ = setChecksumOffload(t.options.Name, unix.ETHTOOL_SRXCSUM)
	}

	err = netlink.LinkSetUp(tunLink)
	if err != nil {
		return err
	}

	if t.options.IPRoute2TableIndex == 0 {
		for {
			t.options.IPRoute2TableIndex = int(rand.Uint32())
			routeList, fErr := netlink.RouteListFiltered(netlink.FAMILY_ALL, &netlink.Route{Table: t.options.IPRoute2TableIndex}, netlink.RT_FILTER_TABLE)
			if len(routeList) == 0 || fErr != nil {
				break
			}
		}
	}

	err = t.setRoute(tunLink)
	if err != nil {
		_ = t.unsetRoute0(tunLink)
		return err
	}

	err = t.unsetRules()
	if err != nil {
		return err
	}
	err = t.setRules()
	if err != nil {
		_ = t.unsetRules()
		return err
	}

	t.setSearchDomainForSystemdResolved()

	return nil
}

func (t *NativeTun) Close() (err error) {
	t.unsetRoute()
	t.unsetRules()
	if t.tunFile != nil {
		err = t.tunFile.Close()
	}
	return
}

func (t *NativeTun) TXChecksumOffload() bool {
	return t.txChecksumOffload
}

func prefixToIPNet(prefix netip.Prefix) *net.IPNet {
	return &net.IPNet{
		IP:   prefix.Addr().AsSlice(),
		Mask: net.CIDRMask(prefix.Bits(), prefix.Addr().BitLen()),
	}
}

func (t *NativeTun) routes(tunLink netlink.Link) ([]netlink.Route, error) {
	routeRanges, err := t.options.BuildAutoRouteRanges()
	if err != nil {
		return nil, err
	}
	routes := make([]netlink.Route, 0, len(routeRanges))
	for _, r := range routeRanges {
		routes = append(routes, netlink.Route{
			Dst:       prefixToIPNet(r),
			LinkIndex: tunLink.Attrs().Index,
			Table:     t.options.IPRoute2TableIndex,
		})
	}
	return routes, nil
}

func (t *NativeTun) nextIndex6() int {
	ruleList, err := netlink.RuleList(netlink.FAMILY_V6)
	if err != nil {
		return -1
	}
	var minIndex int
	for _, rule := range ruleList {
		if rule.Priority > 0 && (minIndex == 0 || rule.Priority < minIndex) {
			minIndex = rule.Priority
		}
	}
	minIndex--
	t.ruleIndex6 = append(t.ruleIndex6, minIndex)
	return minIndex
}

func (t *NativeTun) rules() []*netlink.Rule {
	var p4, p6 bool
	var pRule int
	if len(t.options.Inet4Address) > 0 {
		p4 = true
		pRule += 1
	}
	if len(t.options.Inet6Address) > 0 {
		p6 = true
		pRule += 1
	}
	if pRule == 0 {
		return []*netlink.Rule{}
	}

	var rules []*netlink.Rule
	var it *netlink.Rule

	ruleStart := t.options.IPRoute2RuleIndex
	priority := ruleStart
	priority6 := priority

	nopPriority := ruleStart + 10

	if p4 {
		for _, address := range t.options.Inet4Address {
			it = netlink.NewRule()
			it.Priority = priority
			it.Dst = prefixToIPNet(address.Masked())
			it.Table = t.options.IPRoute2TableIndex
			it.Family = unix.AF_INET
			rules = append(rules, it)
		}
		priority++

		it = netlink.NewRule()
		it.Priority = priority
		it.Table = t.options.IPRoute2TableIndex
		it.SuppressPrefixlen = 0
		it.Family = unix.AF_INET
		rules = append(rules, it)
		priority++
	}
	if p6 {
		it = netlink.NewRule()
		it.Priority = priority6
		it.Table = t.options.IPRoute2TableIndex
		it.SuppressPrefixlen = 0
		it.Family = unix.AF_INET6
		rules = append(rules, it)
		priority6++
	}
	if p4 {
		it = netlink.NewRule()
		it.Priority = priority
		it.Invert = true
		it.Dport = netlink.NewRulePortRange(53, 53)
		it.Table = unix.RT_TABLE_MAIN
		it.SuppressPrefixlen = 0
		it.Family = unix.AF_INET
		rules = append(rules, it)
	}

	if p6 {
		it = netlink.NewRule()
		it.Priority = priority6
		it.Invert = true
		it.Dport = netlink.NewRulePortRange(53, 53)
		it.Table = unix.RT_TABLE_MAIN
		it.SuppressPrefixlen = 0
		it.Family = unix.AF_INET6
		rules = append(rules, it)
	}

	if p4 {
		it = netlink.NewRule()
		it.Priority = priority
		it.IifName = t.options.Name
		it.Goto = nopPriority
		it.Family = unix.AF_INET
		rules = append(rules, it)
		priority++

		it = netlink.NewRule()
		it.Priority = priority
		it.Invert = true
		it.IifName = "lo"
		it.Table = t.options.IPRoute2TableIndex
		it.Family = unix.AF_INET
		rules = append(rules, it)

		it = netlink.NewRule()
		it.Priority = priority
		it.IifName = "lo"
		it.Src = prefixToIPNet(netip.PrefixFrom(netip.IPv4Unspecified(), 32))
		it.Table = t.options.IPRoute2TableIndex
		it.Family = unix.AF_INET
		rules = append(rules, it)

		for _, address := range t.options.Inet4Address {
			it = netlink.NewRule()
			it.Priority = priority
			it.IifName = "lo"
			it.Src = prefixToIPNet(address.Masked())
			it.Table = t.options.IPRoute2TableIndex
			it.Family = unix.AF_INET
			rules = append(rules, it)
		}
		priority++
	}
	if p6 {
		it = netlink.NewRule()
		it.Priority = priority6
		it.IifName = t.options.Name
		it.Goto = nopPriority
		it.Family = unix.AF_INET6
		rules = append(rules, it)

		it = netlink.NewRule()
		it.Priority = priority6
		it.IifName = "lo"
		it.Src = prefixToIPNet(netip.PrefixFrom(netip.IPv6Unspecified(), 1))
		it.Goto = nopPriority
		it.Family = unix.AF_INET6
		rules = append(rules, it)

		it = netlink.NewRule()
		it.Priority = priority6
		it.IifName = "lo"
		it.Src = prefixToIPNet(netip.PrefixFrom(netip.AddrFrom16([16]byte{0: 128}), 1))
		it.Goto = nopPriority
		it.Family = unix.AF_INET6
		rules = append(rules, it)
		priority6++

		for _, address := range t.options.Inet6Address {
			it = netlink.NewRule()
			it.Priority = priority6
			it.IifName = "lo"
			it.Src = prefixToIPNet(address.Masked())
			it.Table = t.options.IPRoute2TableIndex
			it.Family = unix.AF_INET6
			rules = append(rules, it)
		}
		priority6++

		it = netlink.NewRule()
		it.Priority = priority6
		it.Table = t.options.IPRoute2TableIndex
		it.Family = unix.AF_INET6
		rules = append(rules, it)
		priority6++
	}
	if p4 {
		it = netlink.NewRule()
		it.Priority = nopPriority
		it.Family = unix.AF_INET
		rules = append(rules, it)
	}
	if p6 {
		it = netlink.NewRule()
		it.Priority = nopPriority
		it.Family = unix.AF_INET6
		rules = append(rules, it)
	}
	return rules
}

func (t *NativeTun) setRoute(tunLink netlink.Link) error {
	routes, err := t.routes(tunLink)
	if err != nil {
		return err
	}
	for _, route := range routes {
		err := netlink.RouteAdd(&route)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *NativeTun) setRules() error {
	for _, rule := range t.rules() {
		err := netlink.RuleAdd(rule)
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *NativeTun) unsetRoute() error {
	tunLink, err := netlink.LinkByName(t.options.Name)
	if err != nil {
		return err
	}
	return t.unsetRoute0(tunLink)
}

func (t *NativeTun) unsetRoute0(tunLink netlink.Link) error {
	if routes, err := t.routes(tunLink); err == nil {
		for _, route := range routes {
			_ = netlink.RouteDel(&route)
		}
	}
	return nil
}

func (t *NativeTun) unsetRules() error {
	if len(t.ruleIndex6) > 0 {
		for _, index := range t.ruleIndex6 {
			ruleToDel := netlink.NewRule()
			ruleToDel.Family = unix.AF_INET6
			ruleToDel.Priority = index
			err := netlink.RuleDel(ruleToDel)
			if err != nil {
				return err
			}
		}
		t.ruleIndex6 = nil
	}

	ruleList, err := netlink.RuleList(netlink.FAMILY_ALL)
	if err != nil {
		return err
	}
	for _, rule := range ruleList {
		ruleStart := t.options.IPRoute2RuleIndex
		ruleEnd := ruleStart + 10
		if rule.Priority >= ruleStart && rule.Priority <= ruleEnd {
			ruleToDel := netlink.NewRule()
			ruleToDel.Family = rule.Family
			ruleToDel.Priority = rule.Priority
			err = netlink.RuleDel(ruleToDel)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *NativeTun) resetRules() error {
	t.unsetRules()
	return t.setRules()
}

func (t *NativeTun) setSearchDomainForSystemdResolved() {
	ctlPath, err := exec.LookPath("resolvectl")
	if err != nil {
		return
	}
	var dnsServer []netip.Addr
	if len(t.options.Inet4Address) > 0 {
		dnsServer = append(dnsServer, t.options.Inet4Address[0].Addr().Next())
	}
	if len(t.options.Inet6Address) > 0 {
		dnsServer = append(dnsServer, t.options.Inet6Address[0].Addr().Next())
	}
	if len(dnsServer) == 0 {
		return
	}
	dnsServerList := make([]string, 0, len(dnsServer))
	for _, dns := range dnsServer {
		dnsServerList = append(dnsServerList, dns.String())
	}
	go func() {
		_ = execCommand(ctlPath, "domain", t.options.Name, "~.")
		_ = execCommand(ctlPath, "default-route", t.options.Name, "true")
		_ = execCommand(ctlPath, append([]string{"dns", t.options.Name}, dnsServerList...)...)
	}()
}

func execCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Env = os.Environ()
	return command.Run()
}
