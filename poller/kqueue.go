//go:build darwin || netbsd || freebsd || openbsd || dragonfly
// +build darwin netbsd freebsd openbsd dragonfly

package poller

import (
	"fmt"
	"log"
	"syscall"
	"yaev/event"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/5 10:47 上午
 */

type Poll struct {
	fd     int
	change []syscall.Kevent_t
	notes  noteQueue
}

func (p *Poll) Fd() int {
	return p.fd
}

func OpenPoll() *Poll {
	l := new(Poll)
	p, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	l.fd = p
	// To Trigger kqueue
	_, err = syscall.Kevent(l.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
	}}, nil, nil)
	if err != nil {
		panic(err)
	}
	return l
}

func (p *Poll) Close() error {
	return syscall.Close(p.fd)
}

func (p *Poll) Trigger(note interface{}) error {
	p.notes.Add(note)
	_, err := syscall.Kevent(p.fd, []syscall.Kevent_t{{
		Ident:  0,
		Filter: syscall.EVFILT_USER,
		Fflags: syscall.NOTE_TRIGGER,
	}}, nil, nil)
	return err
}

func (p *Poll) AddRead(fd int) {
	p.change = append(p.change, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD,
	})
}

func (p *Poll) AddReadWrite(fd int) {
	fmt.Println("add rw")
	p.change = append(p.change, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD,
	}, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_ADD,
	})
	fmt.Printf("before p.change: %p\n", &(p.change))
}

func (p *Poll) ModRead(fd int) {
	p.change = append(p.change, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_DELETE,
	})
}

func (p *Poll) ModReadWrite(fd int) {
	p.change = append(p.change, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_ADD,
	})
}

func (p *Poll) ModDetach(fd int) {
	p.change = append(p.change, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_DELETE,
	}, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_DELETE,
	})
}

func (p *Poll) Wait(iter func(fd int, note interface{}) error) error {
	events := make([]syscall.Kevent_t, 128)
	for {
		fmt.Printf("%d fd. wait p.change: %p, %v\n", p.fd, &(p.change), p.change)
		n, err := syscall.Kevent(p.fd, p.change, events, nil)
		//fmt.Println("poll fd:", p.fd, ".events:", events[:n])

		if err != nil && err != syscall.EINTR {
			return err
		}
		p.change = p.change[:0]
		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note)
		}); err != nil {
			return err
		}
		for i := 0; i < n; i++ {
			if fd := int(events[i].Ident); fd != 0 {
				if err := iter(fd, nil); err != nil {
					if err == event.ErrKEvent {
						log.Println(events[i])
					} else {
						return err

					}
				}
			}
		}
	}
}
