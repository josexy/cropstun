package bind

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/josexy/cropstun/iface"
	"github.com/josexy/cropstun/route"
)

func TestBindToDeviceForTCP(t *testing.T) {
	dialer := net.Dialer{Timeout: time.Second * 5}
	addr := netip.MustParseAddr("110.242.68.4")

	defaultRoute, err := route.DefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(defaultRoute.InterfaceName, defaultRoute.InterfaceIndex)
	if err := BindToDeviceForConn(defaultRoute.InterfaceName, &dialer, "tcp", addr); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: &http.Transport{DialContext: dialer.DialContext}}
	resp, err := client.Get("http://110.242.68.4:80")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	log.Println(resp.StatusCode)
	log.Println(resp.Header)

	time.Sleep(time.Second * 1)
}

func TestBindToDeviceForUDP(t *testing.T) {
	var lc net.ListenConfig
	defaultRoute, err := route.DefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(defaultRoute.InterfaceName, defaultRoute.InterfaceIndex)

	go func() {
		var lc net.ListenConfig
		conn, _ := lc.ListenPacket(context.Background(), "udp", ":2003")
		defer conn.Close()
		buf := make([]byte, 1024)
		for {
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				break
			}
			conn.WriteTo(buf[:n], addr)
		}
	}()

	addr, err := BindToDeviceForPacket(defaultRoute.InterfaceName, &lc, "udp", "")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := lc.ListenPacket(context.Background(), "udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	time.Sleep(time.Millisecond * 50)
	buf := make([]byte, 1024)

	var targetIP string
	ifaceObj, err := iface.GetInterfaceByName(defaultRoute.InterfaceName)
	if err == nil && ifaceObj != nil && len(ifaceObj.Addrs) > 0 {
		targetIP = ifaceObj.Addrs[0].Addr().String()
	}
	targetAddr, _ := net.ResolveUDPAddr("udp", targetIP+":2003")

	index := 0
	for {
		conn.WriteTo([]byte("hello"), targetAddr)
		n, addr, err := conn.ReadFrom(buf[:])
		log.Println(n, addr, err)
		if err != nil {
			break
		}
		index++
		if index >= 10 {
			break
		}
	}

	time.Sleep(time.Second * 1)
}
