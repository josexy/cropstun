package iface

import (
	"testing"
)

func TestResolveAllInterfaces(t *testing.T) {
	list := GetAllInterfaces()
	for _, iface := range list {
		t.Log(iface.Name, iface.MTU, iface.Index, iface.HardwareAddr)
		for _, addr := range iface.Addrs {
			t.Log("\t->", addr)
		}
	}
}
