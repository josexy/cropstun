package main

import (
	"log"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	T "github.com/josexy/cropstun"
)

var _ T.Handler = (*myHandler)(nil)

type myHandler struct{}

func (*myHandler) HandleTCPConnection(conn net.Conn, info T.Metadata) error {
	log.Printf("tcp, src: %s, dst: %s", info.Src(), info.Dst())
	// do something...
	return nil
}

func (*myHandler) HandleUDPConnection(conn net.PacketConn, info T.Metadata) error {
	log.Printf("udp, src: %s, dst: %s", info.Src(), info.Dst())
	// do something...
	return nil
}

func main() {
	tunOpt := &T.Options{}
	tunIf, err := T.NewTunDevice([]netip.Prefix{netip.MustParsePrefix("198.18.0.1/16")}, tunOpt)
	if err != nil {
		log.Fatal(err)
	}

	stack, err := T.NewStack(T.StackOptions{
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
