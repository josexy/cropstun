package iface

import (
	"bytes"
	"os/exec"
	"strings"
)

// TODO: Deprecated
func DefaultRouteInterface() (string, error) {
	cmd := exec.Command("sh", "-c", "route -n | grep 'UG[ \t]' | awk 'NR==1{print $8}'")
	out := bytes.Buffer{}
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
