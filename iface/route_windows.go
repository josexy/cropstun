package iface

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

var errNotFoundIface = errors.New("not found default gateway interface")
var errNoActiveGateway = errors.New("no active gateway")

// TODO: Deprecated
func DefaultRouteInterface() (string, error) {
	cmd := exec.Command("route", "print", "0.0.0.0")
	out := bytes.Buffer{}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	arr := strings.Split(out.String(), "\n")
	i := 0

	for index, line := range arr {
		if i == 3 {
			if len(arr) < index+2 {
				return "", errNoActiveGateway
			}
			// Network Destination        Netmask          Gateway       Interface  Metric
			// 0.0.0.0 0.0.0.0 192.168.2.1 192.168.2.119 200
			fields := strings.Fields(arr[index+2])
			if len(fields) < 5 {
				return "", errNotFoundIface
			}
			ifaceAddr := fields[3]
			// check all interfaces and obtain one
			for _, iface := range record {
				for _, addr := range iface.Addrs {
					if addr.Addr().String() == ifaceAddr {
						return iface.Name, nil
					}
				}
			}
			break
		}
		if strings.HasPrefix(line, "========") {
			i++
		}
	}
	return "", errNotFoundIface
}
