//go:build linux

package tun

import (
	"github.com/metacubex/gvisor/pkg/tcpip/link/fdbased"
	"github.com/metacubex/gvisor/pkg/tcpip/stack"
)

var _ GVisorTun = (*NativeTun)(nil)

func (t *NativeTun) NewEndpoint() (stack.LinkEndpoint, error) {
	return fdbased.New(&fdbased.Options{
		FDs:               []int{t.tunFd},
		MTU:               t.options.MTU,
		RXChecksumOffload: true,
		TXChecksumOffload: t.txChecksumOffload,
	})
}
