// +build linux

package poller

import (
	"syscall"
	"unsafe"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/5 4:55 下午
 */


type Poll struct {
	fd    int // epoll fd
	wfd   int // wake fd
	notes noteQueue
}

func OpenPoll() *Poll {
	l := new(Poll)
	p, err := syscall.EpollCreate1(0)
	if err != nil {
		panic(err)
	}
	l.fd = p
	r0, _, e0 := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		syscall.Close(p)
		panic(err)
	}
	l.wfd = int(r0)
	l.AddRead(l.wfd)
	return l
}

func (p *Poll) Close() error {
	if err := syscall.Close(p.wfd); err != nil {
		return err
	}
	return syscall.Close(p.fd)
}

func (p *Poll) Trigger(note interface{}) error {
	p.notes.Add(note)
	var x uint64 = 1
	_, err := syscall.Write(p.wfd, (*(*[8]byte)(unsafe.Pointer(&x)))[:])
	return err
}

func (p *Poll) AddRead(fd int) {
	if err:= syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd,
		&syscall.EpollEvent{
			Events: syscall.EPOLLIN,
			Fd: int32(fd),
		}); err != nil{
		panic(err)
	}
}

func (p *Poll) AddReadWrite(fd int) {
	if err:= syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_ADD, fd,
		&syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLOUT,
			Fd: int32(fd),
		}); err != nil{
		panic(err)
	}
}

func (p *Poll) ModRead(fd int) {
	if err:= syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_MOD, fd,
		&syscall.EpollEvent{
			Events: syscall.EPOLLIN,
			Fd: int32(fd),
		}); err != nil{
		panic(err)
	}
}

func (p *Poll) ModReadWrite(fd int) {
	if err:= syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_MOD, fd,
		&syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLOUT,
			Fd: int32(fd),
		}); err != nil{
		panic(err)
	}
}

func (p *Poll) ModDetach(fd int) {
	if err:= syscall.EpollCtl(p.fd, syscall.EPOLL_CTL_DEL, fd,
		&syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLOUT,
			Fd: int32(fd),
		}); err != nil{
		panic(err)
	}
}

func (p *Poll) Wait(iter func(fd int, note interface{}) error) error {
	events := make([]syscall.EpollEvent, 64)
	for  {
		n, err := syscall.EpollWait(p.fd, events, 100)
		if err != nil && err != syscall.EINTR {
			return err
		}
		if err := p.notes.ForEach(func(note interface{}) error {
			return iter(0, note)
		}); err != nil {
			return err
		}
		for i:= 0; i < n; i++ {
			if fd := int(events[i].Fd); fd != p.wfd {
				if err := iter(fd, nil); err != nil {
					return err
				}
			}else {
				var data [8]byte
				syscall.Read(p.wfd, data[:])
			}
		}
	}
}
