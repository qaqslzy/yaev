package eventloop

import (
	"syscall"
	"yaev/event"
	"yaev/yaddr"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/6 4:00 下午
 */

func loopUDPRead(s *server, l *loop, lnidx, fd int) error {
	n, sa, err := syscall.Recvfrom(fd, l.packet, 0)
	if err != nil || n == 0 {
		return nil
	}
	if s.events.Data != nil {
		var sa6 syscall.SockaddrInet6
		switch sa := sa.(type) {
		case *syscall.SockaddrInet4:
			sa6.ZoneId = 0
			sa6.Port = sa.Port
			for i := 0; i < 12; i++ {
				sa6.Addr[i] = 0
			}
			sa6.Addr[12] = sa.Addr[0]
			sa6.Addr[13] = sa.Addr[1]
			sa6.Addr[14] = sa.Addr[2]
			sa6.Addr[15] = sa.Addr[3]
		case *syscall.SockaddrInet6:
			sa6 = *sa
		}
		c := &Connetcion{}
		c.addrIndex = lnidx
		c.localAddr = s.lns[lnidx].Lnaddr
		c.remoteAddr = yaddr.SockaddrToAddr(&sa6)
		in := append([]byte{}, l.packet[:n]...)
		out, action := s.events.Data(c, in)
		if len(out) > 0 {
			if s.events.PreWrite != nil {
				s.events.PreWrite()
			}
			syscall.Sendto(fd, out, 0, sa)
		}
		switch action {
		case event.Shutdown:
			return event.ErrClosing
		}
	}
	return nil
}