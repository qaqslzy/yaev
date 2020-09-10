package listener

import (
	"net"
	"os"
	"syscall"
	"yaev/yaddr"
	"github.com/libp2p/go-reuseport"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/4 4:32 下午
 */

type Listener struct {
	Ln      net.Listener
	Lnaddr  net.Addr
	Pconn   net.PacketConn
	Opts    yaddr.AddrOpts
	F       *os.File
	Fd      int
	Network string
	Addr    string

	stdlibt bool
}

func (ln *Listener) Close() {
	if ln.Fd != 0 {
		_ = syscall.Close(ln.Fd)
	}
	if ln.F != nil {
		_ = ln.F.Close()
	}
	if ln.Ln != nil {
		_ = ln.Ln.Close()
	}
	if ln.Pconn != nil {
		_ = ln.Pconn.Close()
	}
	if ln.Network == "unix" {
		_ = os.RemoveAll(ln.Addr)
	}
}

func NewListener(addr string) (ln *Listener, stdlibt bool, err error) {
	ln = new(Listener)
	ln.Network, ln.Addr, ln.Opts, ln.stdlibt = yaddr.ParseAddr(addr)
	stdlibt = ln.stdlibt
	if ln.Network == "unix" {
		os.RemoveAll(ln.Addr)
	}
	err = ln.listen()
	return
}

func (ln *Listener) listen() (err error) {
	if ln.Network == "udp" {
		if ln.Opts.ReusePort {
			ln.Pconn, err = reuseportListenPacket(ln.Network, ln.Addr)
		} else {
			ln.Pconn, err = net.ListenPacket(ln.Network, ln.Addr)
		}
	} else {
		if ln.Opts.ReusePort {
			ln.Ln, err = reuseportListen(ln.Network, ln.Addr)
		} else {
			ln.Ln, err = net.Listen(ln.Network, ln.Addr)
		}
	}
	if err != nil {
		return
	}
	err = ln.getAddr()
	return
}

func (ln *Listener) getAddr() (err error) {
	if ln.Pconn != nil{
		ln.Lnaddr = ln.Pconn.LocalAddr()
	}else {
		ln.Lnaddr = ln.Ln.Addr()
	}
	// use stdlib on other os which has no epoll or kqueue
	if !ln.stdlibt {
		if err := ln.system(); err != nil {
			return err
		}
	}
	return
}


func (ln *Listener) system() error {
	var err error
	switch netln := ln.Ln.(type) {
	case nil:
		switch pconn := ln.Pconn.(type) {
		case *net.UDPConn:
			ln.F, err = pconn.File()
		}
	case *net.TCPListener:
		ln.F, err = netln.File()
	case *net.UnixListener:
		ln.F, err = netln.File()
	}
	if err != nil {
		ln.Close()
		return err
	}
	ln.Fd = int(ln.F.Fd())
	return syscall.SetNonblock(ln.Fd, true)
}

func reuseportListenPacket(proto, addr string) (l net.PacketConn, err error) {
	return reuseport.ListenPacket(proto, addr)
}

func reuseportListen(proto, addr string) (l net.Listener, err error) {
	return reuseport.Listen(proto, addr)
}
