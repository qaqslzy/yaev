package eventloop

import (
	"sync/atomic"
	"syscall"
	"time"
	"yaev/event"
	"yaev/listener"
	"yaev/poller"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/4 4:12 下午
 */



type loop struct {
	idx     int
	poll    *poller.Poll
	packet  []byte
	fdconns map[int]*Connetcion
	count   int32
}

func (l *loop) Wake(note interface{}) {
	l.poll.Trigger(note)
}

func (l *loop) ModReadWrite(fd int) {
	l.poll.ModReadWrite(fd)
}

func (l *loop) ModRead(fd int) {
	l.poll.ModRead(fd)
}


func loopCloseConn(s *server, l *loop, c *Connetcion, err error) error {
	atomic.AddInt32(&l.count, -1)
	delete(l.fdconns, c.fd)
	return c.Close(s.events, err)
}

func Serve(events event.Events, listeners []*listener.Listener) error {
	s, numLoops, err := newServer(events, listeners)
	defer func() {
		for _, ln := range listeners {
			ln.Close()
		}
	}()

	if err != nil {
		if err == event.ErrShutdown {
			return nil
		}
		return err
	}
	s.serve(numLoops)
	return nil
}

func loopRun(s *server, l *loop) {
	defer func() {
		s.signalShutdown()
		s.wg.Done()
	}()

	if l.idx == 0 && s.events.Tick != nil {
		go loopTicker(s, l)
	}

	l.poll.Wait(func(fd int, note interface{}) error {
		if fd == 0 {
			return loopNote(s, l, note)
		}
		c := l.fdconns[fd]
		switch {
		case c == nil:
			return nil
			//return event.ErrKEvent
			//return loopAccept(s, l, fd)
		case !c.opened:
			return loopOpened(s, c)
		case len(c.out) > 0:
			return loopWrite(s, l, c)
		case c.action != event.None:
			return loopAction(s, l, c)
		default:
			return loopRead(s, l, c)
		}
	})
}

func loopTicker(s *server, l *loop) {
	for {
		if err := l.poll.Trigger(time.Duration(0)); err != nil {
			break
		}
		time.Sleep(<-s.tch)
	}
}

func loopNote(s *server, l *loop, note interface{}) error {
	var err error
	switch v := note.(type) {
	case AddConnFd:
		loopAccept(s, l, int(v))
	case time.Duration:
		delay, action := s.events.Tick()
		switch action {
		case event.None:
		case event.Shutdown:
			err = event.ErrClosing
		}
		s.tch <- delay
	case error:
		err = v
	case *Connetcion:
		if l.fdconns[v.fd] != v {
			return nil
		}
		return loopWake(s, v)
	}
	return err
}

func loopWake(s *server, c *Connetcion) error {
	c.waked(s.events)
	return nil
}

func (l *loop)accept(s *server, fd int) error {
	return loopAccept(s, l, fd)
}

func loopAccept(s *server, l *loop, fd int) error {
	for i, ln := range s.lns {
		if ln.Fd == fd {

			//if err := s.LoadBalance(l); err == NotThatServer {
			//	return nil
			//}

			if ln.Pconn != nil {
				return loopUDPRead(s, l, i, fd)
			}

			if err := NewConnection(fd, i, l); err == nil {
				break
			} else {
				return err
			}
		}
	}

	return nil
}



func loopOpened(s *server, c *Connetcion) error {
	c.Opened(s.events, s.lns)
	return nil
}

func loopWrite(s *server, l *loop, c *Connetcion) error {
	if s.events.PreWrite != nil {
		s.events.PreWrite()
	}
	err := c.write()
	if err != nil {
		return loopCloseConn(s, l, c, err)
	}

	return nil
}

func loopRead(s *server, l *loop, c *Connetcion) error {
	err := c.read(s.events, l)
	if err != nil {
		return loopCloseConn(s, l, c, err)
	}
	return nil
}

func loopAction(s *server, l *loop, c *Connetcion) error {
	switch c.action {
	default:
		c.action = event.None
	case event.Close:
		return loopCloseConn(s, l, c, nil)
	case event.Shutdown:
		return event.ErrClosing
	case event.Detach:
		return loopDetachConn(s, l, c, nil)
	}
	c.readable()
	return nil
}

func loopDetachConn(s *server, l *loop, c *Connetcion, err error) error {
	if s.events.Detached == nil {
		return loopCloseConn(s, l, c, err)
	}

	l.poll.ModDetach(c.fd)

	atomic.AddInt32(&l.count, -1)
	delete(l.fdconns, c.fd)
	if err := syscall.SetNonblock(c.fd, false); err != nil {
		return err
	}
	switch s.events.Detached(c, &detachedConn{fd: c.fd}) {
	case event.None:
	case event.Shutdown:
		return event.ErrClosing
	}
	return nil
}
