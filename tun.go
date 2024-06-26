package tun

import (
	"io"
	"net"
	"net/netip"
	"runtime"
	"strconv"
	"strings"

	N "github.com/josexy/cropstun/common/network"
)

type Metadata struct {
	Source      netip.AddrPort
	Destination netip.AddrPort
}

type TCPConnectionHandler interface {
	HandleTCPConnection(net.Conn, Metadata) error
}

type UDPConnectionHandler interface {
	HandleUDPConnection(net.PacketConn, Metadata) error
}

type Handler interface {
	TCPConnectionHandler
	UDPConnectionHandler
}

type Tun interface {
	io.ReadWriteCloser
	N.VectorisedWriter
	SetupDNS([]netip.Addr) error
	TeardownDNS() error
}

type WinTun interface {
	Tun
	ReadPacket() ([]byte, func(), error)
}

type LinuxTUN interface {
	Tun
	TXChecksumOffload() bool
}

const (
	DefaultMTU                = 9000
	DefaultIPRoute2TableIndex = 4000
	DefaultIPRoute2RuleIndex  = 10086
)

type Options struct {
	Name               string
	Inet4Address       []netip.Prefix
	Inet6Address       []netip.Prefix
	MTU                uint32
	IPRoute2TableIndex int
	IPRoute2RuleIndex  int
	AutoRoute          bool
}

func NewTunDevice(cidrs []netip.Prefix, options *Options) (Tun, error) {
	if options.Name == "" {
		options.Name = CalculateInterfaceName(options.Name)
	}
	if options.MTU == 0 {
		options.MTU = DefaultMTU
	}
	if options.IPRoute2TableIndex <= 0 {
		options.IPRoute2TableIndex = DefaultIPRoute2TableIndex
	}
	if options.IPRoute2RuleIndex <= 0 {
		options.IPRoute2RuleIndex = DefaultIPRoute2RuleIndex
	}
	for _, cidr := range cidrs {
		if cidr.Addr().Is4() {
			options.Inet4Address = append(options.Inet4Address, cidr)
		} else if cidr.Addr().Is6() {
			options.Inet6Address = append(options.Inet6Address, cidr)
		}
	}
	return New(options)
}

func CalculateInterfaceName(name string) (tunName string) {
	if runtime.GOOS == "darwin" {
		tunName = "utun"
	} else if name != "" {
		tunName = name
	} else {
		tunName = "tun"
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	var tunIndex int
	for _, netInterface := range interfaces {
		if strings.HasPrefix(netInterface.Name, tunName) {
			index, parseErr := strconv.ParseInt(netInterface.Name[len(tunName):], 10, 16)
			if parseErr == nil && int(index) >= tunIndex {
				tunIndex = int(index) + 1
			}
		}
	}
	tunName = tunName + strconv.FormatInt(int64(tunIndex), 10)
	return
}
