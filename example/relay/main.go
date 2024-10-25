package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	tun "github.com/josexy/cropstun"
	"github.com/josexy/cropstun/bind"
	"github.com/josexy/cropstun/route"
)

var (
	tunName       string = "tun5" // the tun name for darwin should has the prefix "utun", such as "utun5"
	tunCIDR       string = "198.10.0.10/16"
	outboundIface string
)

var _ tun.Handler = (*myHandler)(nil)

type myHandler struct{}

func tunnel(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	errCh := make(chan error, 2)
	fn := func(dest, src io.ReadWriteCloser) {
		_, err := io.Copy(dest, src)
		errCh <- err
	}
	go fn(dst, src)
	go fn(src, dst)
	<-errCh
}

func (*myHandler) HandleTCPConnection(srcConn net.Conn, info tun.Metadata) error {
	log.Printf("HandleTCPConnection, src: %s, dst: %s, out iface: %s", info.Source, info.Destination, outboundIface)

	// bind an outbound interface to avoid routing loops!!!
	dialer := net.Dialer{Timeout: time.Second * 10}
	if err := bind.BindToDeviceForConn(outboundIface, &dialer); err != nil {
		log.Println(err)
		return err
	}
	dstConn, err := dialer.DialContext(context.Background(), "tcp", info.Destination.String())
	if err != nil {
		log.Println(err)
		return err
	}
	// relay the data between the two connections
	tunnel(dstConn, srcConn)
	return nil
}

func (*myHandler) HandleUDPConnection(conn net.PacketConn, info tun.Metadata) error {
	log.Printf("HandleUDPConnection, src: %s, dst: %s, out iface: %s", info.Source, info.Destination, outboundIface)

	var lc net.ListenConfig
	if err := bind.BindToDeviceForPacket(outboundIface, &lc); err != nil {
		log.Println(err)
		return err
	}
	nm := newNatMap(&lc)
	nm.serve(conn, net.UDPAddrFromAddrPort(info.Destination))
	return nil
}

// go build -o main . && sudo ./main
// curl --interface tun5 www.example.com
func main() {
	rt, err := route.DefaultRouteInterface()
	if err != nil {
		log.Println(err)
	} else {
		outboundIface = rt.InterfaceName
	}

	flag.StringVar(&tunName, "name", tunName, "tun device name")
	flag.StringVar(&tunCIDR, "addr", tunCIDR, "tun device cidr address")
	flag.StringVar(&outboundIface, "outbound-iface", outboundIface, "outbound interface")
	flag.Parse()

	log.Println(tunName, tunCIDR, outboundIface)

	tunOpt := &tun.Options{
		Name:      tunName,
		MTU:       tun.DefaultMTU,
		AutoRoute: true, // AutoRoute will add related routes automatically
	}
	tunIf, err := tun.NewTunDevice([]netip.Prefix{netip.MustParsePrefix(tunCIDR)}, tunOpt)
	if err != nil {
		log.Fatal(err)
	}

	stack, err := tun.NewStack(tun.StackOptions{
		Tun:        tunIf,
		TunOptions: tunOpt,
		Handler:    &myHandler{},
	})
	if err != nil {
		log.Fatal(err)
	}

	if err = stack.Start(); err != nil {
		log.Fatal(err)
	}

	log.Println("running...")
	inter := make(chan os.Signal, 1)
	signal.Notify(inter, syscall.SIGINT)
	<-inter

	stack.Close()
	time.Sleep(time.Second)
	log.Println("done")
}
