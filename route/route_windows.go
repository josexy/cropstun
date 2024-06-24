package route

import (
	"net/netip"

	"github.com/josexy/cropstun/internal/winipcfg"
	"golang.org/x/sys/windows"
)

// zeroTierFakeGatewayIp from
// https://github.com/zerotier/ZeroTierOne/blob/1.8.6/osdep/WindowsEthernetTap.cpp#L994
var zeroTierFakeGatewayIp = netip.MustParseAddr("25.255.255.254")

func DefaultRouteInterface() (*Route, error) {
	rows, err := winipcfg.GetIPForwardTable2(windows.AF_INET)
	if err != nil {
		return nil, err
	}

	lowestMetric := ^uint32(0)
	alias := ""
	var index int

	for _, row := range rows {
		if row.DestinationPrefix.PrefixLength != 0 {
			continue
		}

		if row.NextHop.Addr() == zeroTierFakeGatewayIp {
			continue
		}

		ifrow, err := row.InterfaceLUID.Interface()
		if err != nil || ifrow.OperStatus != winipcfg.IfOperStatusUp {
			continue
		}

		iface, err := row.InterfaceLUID.IPInterface(windows.AF_INET)
		if err != nil {
			continue
		}

		if ifrow.Type == winipcfg.IfTypePropVirtual || ifrow.Type == winipcfg.IfTypeSoftwareLoopback {
			continue
		}

		metric := row.Metric + iface.Metric
		if metric < lowestMetric {
			lowestMetric = metric
			alias = ifrow.Alias()
			index = int(ifrow.InterfaceIndex)
		}
	}

	if alias == "" {
		return nil, ErrNoRoute
	}

	return &Route{
		InterfaceName:  alias,
		InterfaceIndex: index,
	}, nil
}
