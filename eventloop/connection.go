package eventloop

import (
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"yaev/conn"
	"yaev/event"
	"yaev/internal"
	"yaev/listener"
	"yaev/yaddr"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/5 11:45 上午
 */

var connPool = &sync.Pool{
	// 默认的返回值设置，不写这个参数，默认是nil
	New: func() interface{} {
		return new(Connetcion)
	},
}

var ReadZero = errors.New("read zero")

type Connetcion struct {
	fd         int
	lnidx      int
	out        []byte
	sa         syscall.Sockaddr
	reuse      bool
	opened     bool
	action     event.Action
	ctx        interface{}
	addrIndex  int
	localAddr  net.Addr
	remoteAddr net.Addr
	loop       conn.Loop
}

func (c *Connetcion) Context() interface{} { return c.ctx }

func (c *Connetcion) SetContext(ctx interface{}) { c.ctx = ctx }

func (c *Connetcion) AddrIndex() int { return c.addrIndex }

func (c *Connetcion) LocalAddr() net.Addr { return c.localAddr }

func (c *Connetcion) RemoteAddr() net.Addr { return c.remoteAddr }

func (c *Connetcion) Wake() {
	if c.loop != nil {
		c.loop.Wake(c)
	}
}

func (c *Connetcion) Reset() {
	c.fd = -1
	c.lnidx = -1
	c.out = nil
	c.sa = nil
	c.reuse = false
	c.opened = false
	c.action = event.None
	c.ctx = nil
	c.addrIndex = -1
	c.localAddr = nil
	c.remoteAddr = nil
	c.loop = nil
}

func (c *Connetcion) Close(events event.Events, err error) error {
	if err = syscall.Close(c.fd); err != nil {
		log.Println("close", err)
	}
	if events.Closed != nil {
		switch events.Closed(c, err) {
		case event.None:
		case event.Shutdown:
			return event.ErrClosing
		}
	}
	connPool.Put(c)
	return nil
}

func NewConnection(fd, i int, l *loop) error {
	nfd, sa, err := syscall.Accept(fd)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return err
	}
	if err := syscall.SetNonblock(nfd, true); err != nil {
		return err
	}

	c := connPool.Get().(*Connetcion)
	c.Reset()
	c.fd = nfd
	c.lnidx = i
	c.sa = sa
	c.loop = l

	l.fdconns[c.fd] = c
	l.poll.AddReadWrite(c.fd)

	atomic.AddInt32(&l.count, 1)
	return nil
}

func (c *Connetcion) Opened(events event.Events, lns []*listener.Listener) {
	c.opened = true
	c.addrIndex = c.lnidx
	c.localAddr = lns[c.lnidx].Lnaddr
	c.remoteAddr = yaddr.SockaddrToAddr(c.sa)

	if events.Opened != nil {
		out, opts, action := events.Opened(c)
		if len(out) > 0 {
			c.sendMessage(out)
		}

		c.action = action
		c.reuse = opts.ReuseInputBuffer

		if opts.TCPKeepAlive > 0 {
			if _, ok := lns[c.lnidx].Ln.(*net.TCPListener); ok {
				internal.SetKeepAlive(c.fd, int(opts.TCPKeepAlive/time.Second))
			}
		}
	}
	c.readable()
}

func (c *Connetcion) waked(events event.Events) {
	if events.Data == nil {
		return
	}
	out, action := events.Data(c, nil)
	c.action = action
	if len(out) > 0 {
		// TODO 这里可能有 bug append c.out
		c.sendMessage(out)
	}
	c.writeable()
}

func (c *Connetcion) sendMessage(out []byte) {
	if c.out == nil || cap(c.out) > 4096 {
		c.out = append([]byte{}, out...)
	} else {
		c.out = append(c.out[:0], out...)
	}
}

func (c *Connetcion) write() error {
	n, err := syscall.Write(c.fd, c.out)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return err
	}
	if n == len(c.out) {
		if cap(c.out) > 4096 {
			c.out = nil
		} else {
			c.out = c.out[:0]
		}
	} else {
		c.out = c.out[n:]
	}
	c.readable()
	return nil
}

func (c *Connetcion) read(events event.Events, l *loop) error {
	var in []byte
	n, err := syscall.Read(c.fd, l.packet)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return err
	}
	if n == 0 {
		return ReadZero
	}
	in = l.packet[:n]
	if !c.reuse {
		in = append([]byte{}, in...)
	}

	if events.Data != nil {
		out, action := events.Data(c, in)
		c.action = action
		if len(out) > 0 {
			c.sendMessage(out)
		}
	}

	c.writeable()
	return nil

}

func (c *Connetcion) writeable() {
	if len(c.out) != 0 || c.action != event.None {
		c.loop.ModReadWrite(c.fd)
	}
}

func (c *Connetcion) readable() {
	if len(c.out) == 0 && c.action == event.None {
		c.loop.ModRead(c.fd)
	}
}
