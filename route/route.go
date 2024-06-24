package route

import "errors"

var ErrNoRoute = errors.New("no route")

type Route struct {
	InterfaceName  string
	InterfaceIndex int
}
