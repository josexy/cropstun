package iface

import (
	"net/netip"
	"testing"
)

func getDefaultIface(t *testing.T) *Interface {
	ifaceName, err := DefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	iface, err := GetInterfaceByName(ifaceName)
	if err != nil {
		t.Fatal(err)
	}
	return iface
}

func TestResolveAllInterfaces(t *testing.T) {
	for name, iface := range record {
		t.Log(name, iface)
	}
}

func TestDefaultRouteInterface(t *testing.T) {
	iface := getDefaultIface(t)
	t.Log(iface)

	iface, err := GetInterfaceByIndex(iface.Index)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(iface)
}

func TestPickIPv4Addr(t *testing.T) {
	iface := getDefaultIface(t)

	addrs := []string{
		"198.170.10.196",
		"253.79.9.93",
		"221.241.189.203",
		"16.3.228.97",
		"170.65.180.207",
		"22.153.217.89",
		"138.153.168.108",
		"215.73.0.150",
		"190.3.75.199",
		"121.50.82.32",
	}

	for _, addr := range addrs {
		addr := iface.PickIPv4Addr(netip.MustParseAddr(addr))
		t.Log(addr)
	}
}

func TestPickIPv6Addr(t *testing.T) {
	iface := getDefaultIface(t)

	addrs := []string{
		"2DB6:E5E5:7691:8054:279C:EE04:F5FA:A1A9",
		"594A:2C88:5AA9:2057:C1F3:17B3:539C:A030",
		"7793:AC69:472F:402B:F942:B4C9:F084:6E28",
		"7FD2:D335:4295:201B:D39C:632A:92AF:260C",
		"CF8D:DA9E:097D:E267:7FF6:2795:32FF:46DC",
		"FF79:0496:DCCD:87A9:27F2:3076:9BC5:E532",
		"DEBB:DA4E:547A:4F69:93D2:FA25:DD21:41C9",
		"720F:32DB:6A4A:73D8:3C70:5679:6911:8C3D",
		"F829:20C3:AF8C:74BE:FE29:5107:3A66:3E57",
		"3EF8:C409:334C:604E:AB27:FA51:ACF8:63B9",
	}

	for _, addr := range addrs {
		addr := iface.PickIPv6Addr(netip.MustParseAddr(addr))
		t.Log(addr)
	}
}
