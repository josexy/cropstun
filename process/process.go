// referenced from https://github.com/MetaCubeX/mihomo/blob/Alpha/component/process/
package process

import (
	"errors"
	"net/netip"
)

var (
	ErrInvalidNetwork     = errors.New("invalid network")
	ErrPlatformNotSupport = errors.New("not support on this platform")
	ErrNotFound           = errors.New("process not found")
)

const (
	TCP = "tcp"
	UDP = "udp"
)

func FindProcessName(network string, srcIP netip.Addr, srcPort int) (uint32, uint32, string, error) {
	return findProcessName(network, srcIP, srcPort)
}
