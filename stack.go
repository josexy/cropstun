package tun

import (
	"encoding/binary"
	"net"
	"net/netip"
)

type Stack interface {
	Start() error
	Close() error
}

type StackOptions struct {
	Tun        Tun
	TunOptions *Options
	Handler    Handler
}

func NewStack(options StackOptions) (Stack, error) {
	return NewGVisor(options)
}

func BroadcastAddr(inet4Address []netip.Prefix) netip.Addr {
	if len(inet4Address) == 0 {
		return netip.Addr{}
	}
	prefix := inet4Address[0]
	var broadcastAddr [4]byte
	binary.BigEndian.PutUint32(broadcastAddr[:], binary.BigEndian.Uint32(prefix.Masked().Addr().AsSlice())|^binary.BigEndian.Uint32(net.CIDRMask(prefix.Bits(), 32)))
	return netip.AddrFrom4(broadcastAddr)
}
