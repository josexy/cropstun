//go:build darwin || linux || windows

package bufio

import (
	"io"
	"net"
	"syscall"

	"github.com/josexy/cropstun/common"
	"github.com/josexy/cropstun/common/buf"
	N "github.com/josexy/cropstun/common/network"
)

func CreateVectorisedWriter(writer any) (N.VectorisedWriter, bool) {
	switch w := writer.(type) {
	case N.VectorisedWriter:
		return w, true
	case *net.TCPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.UDPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.IPConn:
		return &NetVectorisedWriterWrapper{w}, true
	case *net.UnixConn:
		return &NetVectorisedWriterWrapper{w}, true
	case syscall.Conn:
		rawConn, err := w.SyscallConn()
		if err == nil {
			return &SyscallVectorisedWriter{upstream: writer, rawConn: rawConn}, true
		}
	case syscall.RawConn:
		return &SyscallVectorisedWriter{upstream: writer, rawConn: w}, true
	}
	return nil, false
}

var _ N.VectorisedWriter = (*BufferedVectorisedWriter)(nil)

type BufferedVectorisedWriter struct {
	upstream io.Writer
}

func (w *BufferedVectorisedWriter) WriteVectorised(buffers []*buf.Buffer) error {
	defer buf.ReleaseMulti(buffers)
	bufferLen := buf.LenMulti(buffers)
	if bufferLen == 0 {
		return common.Error(w.upstream.Write(nil))
	} else if len(buffers) == 1 {
		return common.Error(w.upstream.Write(buffers[0].Bytes()))
	}
	var bufferBytes []byte
	if bufferLen > 65535 {
		bufferBytes = make([]byte, bufferLen)
	} else {
		buffer := buf.NewSize(bufferLen)
		defer buffer.Release()
		bufferBytes = buffer.FreeBytes()
	}
	buf.CopyMulti(bufferBytes, buffers)
	return common.Error(w.upstream.Write(bufferBytes))
}

var _ N.VectorisedWriter = (*NetVectorisedWriterWrapper)(nil)

type NetVectorisedWriterWrapper struct {
	upstream io.Writer
}

func (w *NetVectorisedWriterWrapper) WriteVectorised(buffers []*buf.Buffer) error {
	defer buf.ReleaseMulti(buffers)
	netBuffers := net.Buffers(buf.ToSliceMulti(buffers))
	return common.Error(netBuffers.WriteTo(w.upstream))
}

var _ N.VectorisedWriter = (*SyscallVectorisedWriter)(nil)

type SyscallVectorisedWriter struct {
	upstream any
	rawConn  syscall.RawConn
	syscallVectorisedWriterFields
}

func WriteVectorised(writer N.VectorisedWriter, data [][]byte) (n int, err error) {
	var dataLen int
	buffers := make([]*buf.Buffer, 0, len(data))
	for _, p := range data {
		dataLen += len(p)
		buffers = append(buffers, buf.As(p))
	}
	err = writer.WriteVectorised(buffers)
	if err == nil {
		n = dataLen
	}
	return
}
