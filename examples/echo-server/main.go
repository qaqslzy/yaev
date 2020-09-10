package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"yaev"
	"yaev/event"
)

/**
*
* @author Liu Weiyi
* @date 2020/8/7 4:25 上午
 */

func main() {
	var port int
	var loops int
	var udp bool
	var trace bool
	var reuseport bool
	var stdlib bool

	flag.IntVar(&port, "port", 5000, "server port")
	flag.BoolVar(&udp, "udp", false, "listen on udp")
	flag.BoolVar(&reuseport, "reuseport", false, "reuseport (SO_REUSEPORT)")
	flag.BoolVar(&trace, "trace", false, "print packets to console")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.BoolVar(&stdlib, "stdlib", false, "use stdlib")
	flag.Parse()

	s := yaev.NewYServer()
	s.SetNumLoops(loops)
	//s.SetLoadBalance(event.LeastConnections)
	s.Serving(func(srv event.ServerInfo) (action event.Action) {
		log.Printf("echo server started on port %d (loops: %d)", port, srv.NumLoops)
		if reuseport {
			log.Printf("reuseport")
		}
		if stdlib {
			log.Printf("stdlib")
		}
		return
	})

	s.Data(func(c event.Conn, in []byte) (out []byte, action event.Action) {
		if trace {
			log.Printf("%s", strings.TrimSpace(string(in)))
		}
		out = in
		return
	})

	scheme := "tcp"
	if udp {
		scheme = "udp"
	}
	if stdlib {
		scheme += "-net"
	}
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Fatal(s.Start(fmt.Sprintf("%s://:%d?reuseport=%t", scheme, port, reuseport)))
}
