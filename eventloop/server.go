package eventloop

import (
	"errors"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"yaev/event"
	"yaev/listener"
	"yaev/poller"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/5 1:07 下午
 */

var NotThatServer = errors.New("not that server")

type server struct {
	events     event.Events
	loops      []*loop
	lns        []*listener.Listener
	wg         sync.WaitGroup
	cond       *sync.Cond
	balance    event.LoadBalance
	accepted   uintptr
	tch        chan time.Duration
	listenLoop *ListenLoop

	loopsNum   int
	nextLoop   int
}

func newServer(events event.Events, listeners []*listener.Listener) (s *server, numLoops int, err error) {
	numLoops = events.NumLoops
	if numLoops <= 0 {
		if numLoops == 0 {
			numLoops = 1
		} else {
			numLoops = runtime.NumCPU()
		}
	}
	s = new(server)
	s.events = events
	s.loopsNum = numLoops
	s.nextLoop = 0
	s.lns = listeners
	s.cond = sync.NewCond(&sync.Mutex{})
	s.balance = events.LoadBalance
	s.tch = make(chan time.Duration)

	err = s.Serving(numLoops)

	return
}

func (s *server) serve(numLoops int) {
	defer func() {
		s.waitForShutdown()
		s.closeTrigger()
		s.wg.Wait()

		s.close()
	}()
	s.createLoops(numLoops)

	s.listenLoop = NewListenLoop(s)
	go s.listenLoop.listen()

	s.wg.Add(numLoops)
	for _, l := range s.loops {
		go loopRun(s, l)
	}
}
func (s *server) waitForShutdown() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

func (s *server) signalShutdown() {
	s.cond.L.Lock()
	s.cond.Signal()
	s.cond.L.Unlock()
}

func (s *server) closeTrigger() {
	for _, l := range s.loops {
		_ = l.poll.Trigger(event.ErrClosing)
	}
	_ = s.listenLoop.poll.Trigger(event.ErrClosing)
}

func (s *server) close() {
	for _, l := range s.loops {
		for _, c := range l.fdconns {
			_ = loopCloseConn(s, l, c, nil)
		}
		_ = l.poll.Close()
	}
}

func (s *server) Serving(numLoops int) error {
	if s.events.Serving != nil {
		var svr event.ServerInfo
		svr.NumLoops = numLoops
		svr.Addrs = make([]net.Addr, len(s.lns))
		for i, ln := range s.lns {
			svr.Addrs[i] = ln.Lnaddr
		}
		action := s.events.Serving(svr)
		switch action {
		case event.None:
		case event.Shutdown:
			return event.ErrShutdown
		}
	}
	return nil
}
func (s *server) createLoops(numLoops int) {
	for i := 0; i < numLoops; i++ {
		var l = &loop{
			idx:     i,
			poll:    poller.OpenPoll(),
			packet:  make([]byte, 0xFFFF),
			fdconns: make(map[int]*Connetcion),
		}
		// TODO 惊群
		//for _, ln := range s.lns {
		//	l.poll.AddRead(ln.Fd)
		//}
		s.loops = append(s.loops, l)
	}
}

func (s *server) LoadBalance(l *loop) error {
	if len(s.loops) > 1 {
		switch s.balance {
		case event.LeastConnections:
			n := atomic.LoadInt32(&l.count)
			for _, lp := range s.loops {
				if lp.idx != l.idx {
					if atomic.LoadInt32(&lp.count) < n {
						return NotThatServer
					}
				}
			}
		case event.RoundRobin:
			idx := int(atomic.LoadUintptr(&s.accepted)) % len(s.loops)
			if idx != l.idx {
				return NotThatServer
			}
			atomic.AddUintptr(&s.accepted, 1)

		}
	}
	return nil
}
