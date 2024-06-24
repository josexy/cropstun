//go:build darwin || linux

package bufio

import (
	"os"
	"sync"
	"syscall"
	"unsafe"

	"github.com/josexy/cropstun/common/buf"

	"golang.org/x/sys/unix"
)

type syscallVectorisedWriterFields struct {
	access    sync.Mutex
	iovecList *[]unix.Iovec
}

func (w *SyscallVectorisedWriter) WriteVectorised(buffers []*buf.Buffer) error {
	w.access.Lock()
	defer w.access.Unlock()
	defer buf.ReleaseMulti(buffers)
	var iovecList []unix.Iovec
	if w.iovecList != nil {
		iovecList = *w.iovecList
	}
	iovecList = iovecList[:0]
	for index, buffer := range buffers {
		iovecList = append(iovecList, unix.Iovec{Base: &buffer.Bytes()[0]})
		iovecList[index].SetLen(buffer.Len())
	}
	if w.iovecList == nil {
		w.iovecList = new([]unix.Iovec)
	}
	*w.iovecList = iovecList // cache
	var innerErr unix.Errno
	err := w.rawConn.Write(func(fd uintptr) (done bool) {
		//nolint:staticcheck
		//goland:noinspection GoDeprecation
		_, _, innerErr = unix.Syscall(syscall.SYS_WRITEV, fd, uintptr(unsafe.Pointer(&iovecList[0])), uintptr(len(iovecList)))
		return innerErr != unix.EAGAIN && innerErr != unix.EWOULDBLOCK
	})
	if innerErr != 0 {
		err = os.NewSyscallError("SYS_WRITEV", innerErr)
	}
	for index := range iovecList {
		iovecList[index] = unix.Iovec{}
	}
	return err
}
