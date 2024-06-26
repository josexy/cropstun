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
	tunName    string = "utun5"
	tunCIDR    string = "192.168.0.10/16"
	remoteAddr string = ""
)

var _ tun.Handler = (*myHandler)(nil)

type myHandler struct{}

func tunnel(dst, src io.ReadWriteCloser) {
	errCh := make(chan error, 2)
	defer dst.Close()
	defer src.Close()
	fn := func(dest, src io.ReadWriteCloser) {
		_, err := io.Copy(dest, src)
		errCh <- err
	}
	go fn(dst, src)
	go fn(src, dst)
	<-errCh
}

func (*myHandler) HandleTCPConnection(conn net.Conn, info tun.Metadata) error {
	log.Printf("tcp, src: %s, dst: %s", info.Source, info.Destination)
	dialer := net.Dialer{Timeout: time.Second * 10}
	defaultRoute, err := route.DefaultRouteInterface()
	if err != nil {
		log.Println(err)
		return err
	}
	// bind an outbound interface to avoid routing loops
	if err := bind.BindToDeviceForConn(defaultRoute.InterfaceName, &dialer, "tcp4", info.Destination.Addr()); err != nil {
		log.Println(err)
		return err
	}
	target, err := dialer.DialContext(context.Background(), "tcp4", remoteAddr)
	if err != nil {
		log.Println(err)
		return err
	}
	tunnel(target, conn)
	return nil
}

func (*myHandler) HandleUDPConnection(conn net.PacketConn, info tun.Metadata) error {
	log.Printf("udp, src: %s, dst: %s", info.Source, info.Destination)
	// do something...
	return nil
}

func main() {
	flag.StringVar(&tunName, "name", tunName, "tun device name")
	flag.StringVar(&tunCIDR, "addr", tunCIDR, "tun device cidr address")
	flag.StringVar(&remoteAddr, "remote", remoteAddr, "test remote address")
	flag.Parse()

	if remoteAddr == "" {
		log.Fatal("need remote address")
	}

	log.Println(tunName, tunCIDR, remoteAddr)

	tunOpt := &tun.Options{
		Name:      tunName,
		MTU:       tun.DefaultMTU,
		AutoRoute: true,
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
