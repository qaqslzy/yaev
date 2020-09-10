package event

import (
	"errors"
	"io"
	"net"
	"time"
	"yaev/conn"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/4 4:15 下午
 */

var ErrShutdown = errors.New("shutdown")
var ErrClosing = errors.New("closing")
var ErrCloseConns = errors.New("close conns")
var ErrKEvent = errors.New("wrong kevent")

type Conn conn.Conn
type Action int
type LoadBalance int

const (
	None Action = iota
	// detach a connection. Not available for UDP
	Detach
	// close the connection
	Close
	// shutdown the server
	Shutdown
)

const (
	Random LoadBalance = iota

	RoundRobin

	LeastConnections
)

type Options struct {
	// SO_KEEPALIVE socket option
	TCPKeepAlive time.Duration
	// reuse buffer
	ReuseInputBuffer bool
}

type ServerInfo struct {
	Addrs []net.Addr
	NumLoops int
}

type Events struct {
	NumLoops    int
	LoadBalance LoadBalance

	Serving func(server ServerInfo) (action Action)

	Opened func(c Conn) (out []byte, opts Options, action Action)

	Closed func(c Conn, err error) (action Action)

	Detached func(c Conn, rwc io.ReadWriteCloser) (action Action)

	PreWrite func()

	Data func(c Conn, in []byte) (out []byte, action Action)

	Tick func() (delay time.Duration, action Action)
}

