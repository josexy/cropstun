package tun

type Stack interface {
	Start() error
	Close() error
}

type StackOptions struct {
	Tun        Tun
	TunOptions *Options
	Handler    Handler
}

func NewStack(options StackOptions) (Stack, error) {
	return newGVisor(options)
}
