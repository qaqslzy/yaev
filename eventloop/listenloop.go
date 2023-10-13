package eventloop

import (
	"fmt"
	"yaev/event"
	"yaev/poller"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/7 12:20 上午
 */

type ListenLoop struct {
	poll *poller.Poll
	srv  *server
}

type AddConnFd int

func NewListenLoop(srv *server) (ll *ListenLoop) {
	ll = new(ListenLoop)
	ll.srv = srv
	ll.poll = poller.OpenPoll()
	for _, ln := range srv.lns {
		ll.poll.AddRead(ln.Fd)
	}
	return
}

func (ll *ListenLoop) close() {
	ll.poll.Close()
}

func (ll *ListenLoop) listen() {
	defer func() {
		ll.srv.signalShutdown()
		ll.close()
	}()
	fmt.Println("ll fd:", ll.poll.Fd())

	ll.poll.Wait(func(fd int, note interface{}) error {
		fmt.Println("ll.wait.fd:", fd)
		if fd == 0 {
			return listenLoopNote(note)
		}
		iol := ll.pickIOLoop()

		for i, ln := range ll.srv.lns {
			if ln.Fd == fd {

				//if err := s.LoadBalance(l); err == NotThatServer {
				//	return nil
				//}

				if ln.Pconn != nil {
					return loopUDPRead(ll.srv, iol, i, fd)
				}

				if nfd, err := NewConnection(fd, i, iol); err == nil {
					return iol.poll.Trigger(AddConnFd(nfd))
				} else {
					return err
				}
			}
		}
		return nil
		//return iol.poll.Trigger(AddConnFd(fd))
	})
}

func (ll *ListenLoop) pickIOLoop() (l *loop) {

	if ll.srv.balance == event.Random {
		l = ll.srv.loops[ll.srv.nextLoop]
		ll.srv.nextLoop = (ll.srv.nextLoop + 1) % ll.srv.loopsNum
		return
	}
	for _, l := range ll.srv.loops {
		if err := ll.srv.LoadBalance(l); err == nil {
			return l
		}
	}
	return nil
}

func listenLoopNote(note interface{}) error {
	switch v := note.(type) {
	case error:
		return v
	}
	return nil
}
