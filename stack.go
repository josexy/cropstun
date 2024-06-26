package tun

type Stack interface {
	Start() error
	Close() error
	TunDevice() Tun
}

type StackOptions struct {
	Tun        Tun
	TunOptions *Options
	Handler    Handler
}

func NewStack(options StackOptions) (Stack, error) {
	return newGVisor(options)
}
