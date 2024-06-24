# cropped for sing-tun

This project is based on the [sing-tun](https://github.com/SagerNet/sing-tun) project, with a lot of cuts and streamlining. Only the `gVisor` mode has been retained and made more generic.

Due to the removal of many customizations and some referenced code, it is for personal use only, and only supports Linux, macOS and Windows.

# usage

```shell
go get github.com/josexy/cropstun
```

Here is a simple example:

```go
type myHandler struct{}

func (*myHandler) HandleTCPConnection(conn net.Conn, info tun.Metadata) error {
	log.Printf("tcp, src: %s, dst: %s", info.Source, info.Destination)
	// do something...
	return nil
}

func (*myHandler) HandleUDPConnection(conn net.PacketConn, info tun.Metadata) error {
	log.Printf("udp, src: %s, dst: %s", info.Source, info.Destination)
	// do something...
	return nil
}

func main() {
	tunOpt := new(tun.Options)
	tunIf, err := tun.NewTunDevice([]netip.Prefix{netip.MustParsePrefix("198.18.0.1/16")}, tunOpt)
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
	inter := make(chan os.Signal, 1)
	signal.Notify(inter, syscall.SIGINT)
	<-inter
	stack.Close()
	time.Sleep(time.Second)
}
```

## License

```
Copyright (C) 2022 by nekohasekai <contact-sagernet@sekai.icu>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
```