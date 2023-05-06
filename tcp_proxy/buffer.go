package tcpproxy

import (
	"io"
	"fmt"
	"errors"
	"encoding/hex"
)

var errInvalidWrite = errors.New("invalid write result")

// copyBuffer same as the original builtin golang function
// with some customizations
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
    if buf == nil {
        size := 32 * 1024
        if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
            if l.N < 1 {
                size = 1
            } else {
                size = int(l.N)
            }
        }
        buf = make([]byte, size)
    }
    for {
        nr, er := src.Read(buf)
        if nr > 0 {
						fmt.Printf("copyBuffer: \n%v\n", hex.Dump(buf[0:nr]))
            nw, ew := dst.Write(buf[0:nr])
            if nw < 0 || nr < nw {
                nw = 0
                if ew == nil {
                    ew = errInvalidWrite
                }
            }
            written += int64(nw)
            if ew != nil {
                err = ew
                break
            }
            if nr != nw {
                err = io.ErrShortWrite
                break
            }
        }
        if er != nil {
            if er != io.EOF {
                err = er
            }
            break
        }
    }
    return written, err
}
