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
)

func TestBindToDeviceForTCP(t *testing.T) {
	dialer := net.Dialer{Timeout: time.Second * 5}
	addr := netip.MustParseAddr("110.242.68.4")

	ifaceName, err := iface.DefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(ifaceName)
	if err := BindToDeviceForConn(ifaceName, &dialer, "tcp4", addr); err != nil {
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

	time.Sleep(time.Second * 5)
}

func TestBindToDeviceForUDP(t *testing.T) {
	var lc net.ListenConfig
	ifaceName, err := iface.DefaultRouteInterface()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(ifaceName)

	addr, err := BindToDeviceForPacket(ifaceName, &lc, "udp4", "")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := lc.ListenPacket(context.Background(), "udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	buf := make([]byte, 1024)
	targetAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2003")
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

	time.Sleep(time.Second * 5)
}
