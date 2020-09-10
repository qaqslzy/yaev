package conn

import "net"

/**
*
* @author Liu Weiyi
* @date 2020/8/4 4:16 下午
 */

type Conn interface {
	Context() interface{}
	// set a user-defined context
	SetContext(interface{})
	// index of server address
	AddrIndex() int

	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	// wake triggers a data event
	Wake()
}

type Loop interface {
	Wake(interface{})
	ModReadWrite(int)
	ModRead(int)
}