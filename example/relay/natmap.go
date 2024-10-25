package main

import (
	"context"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"
)

const udpPacketTimeout = 30 * time.Second

type endpoint struct {
	addrPort netip.AddrPort
	srcAddr  net.Addr
	srcConn  net.PacketConn
	dstConn  net.PacketConn
}

type natmap struct {
	lock     sync.RWMutex
	natCache map[netip.AddrPort]*endpoint
	lc       *net.ListenConfig
}

func newNatMap(lc *net.ListenConfig) *natmap {
	return &natmap{
		lc:       lc,
		natCache: make(map[netip.AddrPort]*endpoint),
	}
}

func (nat *natmap) serve(conn net.PacketConn, dstAddr net.Addr) error {
	defer func() {
		conn.Close()
		nat.lock.Lock()
		for _, ep := range nat.natCache {
			ep.dstConn.Close()
		}
		clear(nat.natCache)
		nat.lock.Unlock()
	}()

	handler := func(ep *endpoint) {
		defer func() {
			ep.dstConn.Close()
			nat.lock.Lock()
			delete(nat.natCache, ep.addrPort)
			nat.lock.Unlock()
		}()
		buf := make([]byte, 10240)
		for {
			ep.dstConn.SetReadDeadline(time.Now().Add(udpPacketTimeout))
			n, srcAddr, err := ep.dstConn.ReadFrom(buf)
			if err != nil {
				log.Println(err)
				return
			}
			if srcAddr != nil && dstAddr != nil && srcAddr.String() != dstAddr.String() {
				continue
			}
			ep.srcConn.WriteTo(buf[:n], ep.srcAddr)
		}
	}

	buf := make([]byte, 10240)
	for {
		conn.SetReadDeadline(time.Now().Add(udpPacketTimeout))
		n, srcAddr, err := conn.ReadFrom(buf)
		if err != nil {
			log.Println(err)
			return err
		}
		srcUdpAddr, ok := srcAddr.(*net.UDPAddr)
		if !ok {
			continue
		}
		addr, ok := netip.AddrFromSlice(srcUdpAddr.IP)
		if !ok {
			continue
		}
		addrPort := netip.AddrPortFrom(addr, uint16(srcUdpAddr.Port))

		var ep *endpoint
		nat.lock.RLock()
		if ep, ok = nat.natCache[addrPort]; !ok {
			nat.lock.RUnlock()
			dstconn, err := nat.lc.ListenPacket(context.Background(), "udp", "")
			if err != nil {
				continue
			}
			ep = &endpoint{
				srcAddr:  srcAddr,
				addrPort: addrPort,
				srcConn:  conn,
				dstConn:  dstconn,
			}
			nat.lock.Lock()
			nat.natCache[addrPort] = ep
			nat.lock.Unlock()
			go handler(ep)
		} else {
			nat.lock.RUnlock()
		}

		ep.dstConn.WriteTo(buf[:n], dstAddr)
	}
}
