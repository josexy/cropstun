package network

import (
	"github.com/josexy/cropstun/common/buf"
)

type VectorisedWriter interface {
	WriteVectorised(buffers []*buf.Buffer) error
}
