package eventloop

import (
	"io"
	"syscall"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/5 3:27 下午
 */

type detachedConn struct {
	fd int
}

func (c *detachedConn) Close() error {
	err := syscall.Close(c.fd)
	if err != nil {
		return err
	}
	c.fd = -1
	return nil
}

func (c *detachedConn) Read(p []byte) (n int, err error) {
	n, err = syscall.Read(c.fd, p)
	if err != nil {
		return n, err
	}
	if n == 0 {
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	return n, nil
}

func (c *detachedConn) Write(p []byte) (n int, err error) {
	n = len(p)
	for len(p) > 0 {
		nn, err := syscall.Write(c.fd, p)
		if err != nil {
			return n, err
		}
		p = p[nn:]
	}
	return n, nil
}
