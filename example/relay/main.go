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

type myHandler struct {
	dialer *net.Dialer
	lc     *net.ListenConfig
}

func tunnelTCP(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	errCh := make(chan error, 2)
	fn := func(dest, src io.ReadWriteCloser) {
		_, err := io.Copy(dest, src)
		if err != nil {
			log.Println("tunnel tcp, err:", err)
		}
		errCh <- err
	}
	go fn(dst, src)
	go fn(src, dst)
	<-errCh
}

func tunnelUDP(dst, src net.PacketConn, to net.Addr, timeout time.Duration) {
	defer dst.Close()
	defer src.Close()
	errCh := make(chan error, 2)
	fn := func(dest, src net.PacketConn, to net.Addr, timeout time.Duration) {
		err := copyPacketData(dest, src, to, timeout)
		if err != nil {
			log.Println("tunnel udp, err:", err)
		}
		errCh <- err
	}
	go fn(dst, src, to, timeout)
	go fn(src, dst, nil, timeout)
	<-errCh
}

func copyPacketData(dst, src net.PacketConn, to net.Addr, timeout time.Duration) error {
	buf := make([]byte, 10240)
	for {
		src.SetReadDeadline(time.Now().Add(timeout))
		n, from, err := src.ReadFrom(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if to == nil {
			to = from
		}
		if _, err = dst.WriteTo(buf[:n], to); err != nil {
			return err
		}
	}
}

type symmetricNATPacketConn struct {
	net.PacketConn
	dst string
	src net.Addr
}

func newSymmetricNATPacketConn(pc net.PacketConn, metadata tun.Metadata) *symmetricNATPacketConn {
	return &symmetricNATPacketConn{
		PacketConn: pc,
		dst:        metadata.Destination.String(),
		src:        net.UDPAddrFromAddrPort(metadata.Source),
	}
}

func (pc *symmetricNATPacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	for {
		n, from, err := pc.PacketConn.ReadFrom(p)
		if from != nil && from.String() != pc.dst {
			log.Printf("symmetric NAT %s->%s: drop packet from %s", pc.src.String(), pc.dst, from)
			continue
		}
		return n, pc.src, err
	}
}

func (h *myHandler) HandleTCPConnection(srcConn tun.TCPConn, info tun.Metadata) error {
	log.Printf("HandleTCPConnection, src: %s, dst: %s, out iface: %s", info.Source, info.Destination, outboundIface)
	dstConn, err := h.dialer.DialContext(context.Background(), "tcp", info.Destination.String())
	if err != nil {
		log.Println(err)
		return err
	}
	// relay the data between the two connections
	tunnelTCP(dstConn, srcConn)
	return nil
}

func (h *myHandler) HandleUDPConnection(srcConn tun.UDPConn, info tun.Metadata) error {
	log.Printf("HandleUDPConnection, src: %s, dst: %s, out iface: %s", info.Source, info.Destination, outboundIface)

	dstConn, err := h.lc.ListenPacket(context.Background(), "udp", "")
	if err != nil {
		log.Println(err)
		return err
	}
	dstConn = newSymmetricNATPacketConn(dstConn, info)
	tunnelUDP(dstConn, srcConn, net.UDPAddrFromAddrPort(info.Destination), time.Second*10)
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
	// tunIf.SetupDNS([]netip.Addr{netip.MustParseAddr("114.114.114.114")})

	handler := &myHandler{
		dialer: &net.Dialer{Timeout: time.Second * 10},
		lc:     &net.ListenConfig{},
	}
	// bind an outbound interface to avoid routing loops!!!
	if err := bind.BindToDeviceForConn(outboundIface, handler.dialer); err != nil {
		log.Fatal(err)
	}
	if err := bind.BindToDeviceForPacket(outboundIface, handler.lc); err != nil {
		log.Fatal(err)
	}

	stack, err := tun.NewStack(tun.StackOptions{
		Tun:        tunIf,
		TunOptions: tunOpt,
		Handler:    handler,
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
