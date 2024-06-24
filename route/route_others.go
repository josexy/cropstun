//go:build !(darwin || linux || windows)

package route

import (
	"os"
)

func DefaultRouteInterface() (*Route, error) {
	return nil, os.ErrInvalid
}
