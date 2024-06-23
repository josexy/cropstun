//go:build linux

package tun

import (
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type ifreqData struct {
	ifrName [unix.IFNAMSIZ]byte
	ifrData uintptr
}

type ethtoolValue struct {
	cmd  uint32
	data uint32
}

//go:linkname ioctlPtr golang.org/x/sys/unix.ioctlPtr
func ioctlPtr(fd int, req uint, arg unsafe.Pointer) (err error)

func checkChecksumOffload(name string, cmd uint32) (bool, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_IP)
	if err != nil {
		return false, err
	}
	defer syscall.Close(fd)
	ifr := ifreqData{}
	copy(ifr.ifrName[:], name)
	data := ethtoolValue{cmd: cmd}
	ifr.ifrData = uintptr(unsafe.Pointer(&data))
	err = ioctlPtr(fd, unix.SIOCETHTOOL, unsafe.Pointer(&ifr))
	if err != nil {
		return false, os.NewSyscallError("SIOCETHTOOL ETHTOOL_GTXCSUM", err)
	}
	return data.data == 0, nil
}

func setChecksumOffload(name string, cmd uint32) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_IP)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)
	ifr := ifreqData{}
	copy(ifr.ifrName[:], name)
	data := ethtoolValue{cmd: cmd, data: 0}
	ifr.ifrData = uintptr(unsafe.Pointer(&data))
	err = ioctlPtr(fd, unix.SIOCETHTOOL, unsafe.Pointer(&ifr))
	if err != nil {
		return os.NewSyscallError("SIOCETHTOOL ETHTOOL_STXCSUM", err)
	}
	return nil
}
