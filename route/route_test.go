package route

import (
	"testing"

	"github.com/josexy/cropstun/iface"
)

func TestDefaultRouteInterface(t *testing.T) {
	route, err := DefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("name:", route.InterfaceName, "index:", route.InterfaceIndex)
	t.Log(iface.GetInterfaceByIndex(route.InterfaceIndex))
}
