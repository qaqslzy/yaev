package yaev

import (
	"io"
	"math/rand"
	"net"
	"time"
	"yaev/event"
	"yaev/eventloop"
	"yaev/listener"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/4 3:49 下午
 */

func init()  {
	rand.Seed(time.Now().Unix())
}

// InputStream is a helper type for managing input streams from inside
// the Data event.
type InputStream struct{ b []byte }

// Begin accepts a new packet and returns a working sequence of
// unprocessed bytes.
func (is *InputStream) Begin(packet []byte) (data []byte) {
	if len(is.b) > 0 {
		is.b = append(is.b, packet...)
		data = is.b
	} else {
		data = packet
	}

	return data
}

// End shifts the stream to match the unprocessed data.
func (is *InputStream) End(data []byte) {
	if len(data) > 0 {
		if len(data) != len(is.b) {
			is.b = append(is.b[:0], data...)
		}
	} else if len(is.b) > 0 {
		is.b = is.b[:0]
	}
}

type YServer struct {
	Addrs []net.Addr
	// the number of loops that the server is using
	numLoops    int
	loadBalance event.LoadBalance
	events      event.Events
	addrsStr    []string
}

func (s *YServer) ClearAddrs() {
	s.Addrs = nil
	s.addrsStr = nil
}
func (s *YServer) SetLoadBalance(loadBalance event.LoadBalance) {
	s.loadBalance = loadBalance
}

func (s *YServer) SetNumLoops(numLoops int) {
	s.numLoops = numLoops
}

func NewYServer() (s *YServer) {
	s = new(YServer)
	s.events = event.Events{}
	s.Addrs = []net.Addr{}
	s.addrsStr = []string{}
	return
}

func (s *YServer) Serving(f func(server event.ServerInfo) (action event.Action)) {
	s.events.Serving = f
}

func (s *YServer) Opened(f func(c event.Conn) (out []byte, opts event.Options, action event.Action)) {
	s.events.Opened = f
}

func (s *YServer) Closed(f func(c event.Conn, err error) (action event.Action)) {
	s.events.Closed = f
}

func (s *YServer) Detached(f func(c event.Conn, rwc io.ReadWriteCloser) (action event.Action)) {
	s.events.Detached = f
}

func (s *YServer) PreWrite(f func()) {
	s.events.PreWrite = f
}

func (s *YServer) Data(f func(c event.Conn, in []byte) (out []byte, action event.Action)) {
	s.events.Data = f
}

func (s *YServer) Tick(f func() (delay time.Duration, action event.Action)) {
	s.events.Tick = f
}

func (s *YServer) SetAddr(addr ...string) {
	if s.addrsStr == nil {
		s.addrsStr = []string{}
	}
	for _, addr := range addr {
		s.addrsStr = append(s.addrsStr, addr)
	}
}

func (s *YServer) Start(addr ...string) error {
	s.SetAddr(addr...)
	s.events.NumLoops = s.numLoops
	s.events.LoadBalance = s.loadBalance
	var lns []*listener.Listener
	stdlib := false
	for _, addr := range s.addrsStr {
		ln, stdlibt, err := listener.NewListener(addr)
		if err != nil {
			return err
		}
		if ln != nil {
			lns = append(lns, ln)
		}
		if stdlibt {
			stdlib = true
		}
	}
	if stdlib {
		// TODO stdserver()
	}

	return eventloop.Serve(s.events, lns)
}
