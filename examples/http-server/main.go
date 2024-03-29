// Copyright 2017 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"yaev"
	"yaev/event"
)

var res string

type request struct {
	proto, method string
	path, query   string
	head, body    string
	remoteAddr    string
}

func main() {
	var port int
	var loops int
	var aaaa bool
	var noparse bool
	var unixsocket string
	var stdlib bool

	flag.StringVar(&unixsocket, "unixsocket", "", "unix socket")
	flag.IntVar(&port, "port", 8250, "server port")
	flag.BoolVar(&aaaa, "aaaa", false, "aaaaa....")
	flag.BoolVar(&noparse, "noparse", true, "do not parse requests")
	flag.BoolVar(&stdlib, "stdlib", false, "use stdlib")
	flag.IntVar(&loops, "loops", -1, "num loops")
	flag.Parse()

	if os.Getenv("NOPARSE") == "1" {
		noparse = true
	}

	if aaaa {
		res = strings.Repeat("a", 1024)
	} else {
		res = "Hello Yaev!\r\n"
	}

	server := yaev.NewYServer()
	server.SetNumLoops(loops)
	server.SetLoadBalance(event.LeastConnections)
	server.Serving(func(srv event.ServerInfo) (action event.Action) {
		log.Printf("http server started on port %d (loops: %d)", port, srv.NumLoops)
		if unixsocket != "" {
			log.Printf("http server started at %s", unixsocket)
		}
		if stdlib {
			log.Printf("stdlib")
		}
		return
	})

	server.Opened(func(c event.Conn) (out []byte, opts event.Options, action event.Action) {
		c.SetContext(&yaev.InputStream{})
		opts.ReuseInputBuffer = true
		//log.Printf("opened: laddr: %v: raddr: %v", c.LocalAddr(), c.RemoteAddr())
		return
	})

	server.Closed(func(c event.Conn, err error) (action event.Action) {
		//log.Printf("closed: %s: %s", c.LocalAddr().String(), c.RemoteAddr().String())
		return
	})
	server.Data(func(c event.Conn, in []byte) (out []byte, action event.Action) {
		if in == nil {
			return
		}
		is := c.Context().(*yaev.InputStream)
		data := is.Begin(in)
		if noparse && bytes.Contains(data, []byte("\r\n\r\n")) {
			// for testing minimal single packet request -> response.
			out = appendresp(nil, "200 OK", "", res)
			return
		}
		// process the pipeline
		var req request
		for {
			leftover, err := parsereq(data, &req)
			if err != nil {
				// bad thing happened
				out = appendresp(out, "500 Error", "", err.Error()+"\n")
				action = event.Close
				break
			} else if len(leftover) == len(data) {
				// request not ready, yet
				break
			}
			// handle the request
			req.remoteAddr = c.RemoteAddr().String()
			out = appendhandle(out, &req)
			data = leftover
		}
		is.End(data)
		return
	})

	var ssuf string
	if stdlib {
		ssuf = "-net"
	}
	// We at least want the single http address.
	addrs := []string{fmt.Sprintf("tcp"+ssuf+"://0.0.0.0:%d", port)}
	if unixsocket != "" {
		addrs = append(addrs, fmt.Sprintf("unix"+ssuf+"://%s", unixsocket))
	}
	// Start serving!
	err := server.Start(addrs...)
	if err != nil {
		panic(err)
	}

}

// appendhandle handles the incoming request and appends the response to
// the provided bytes, which is then returned to the caller.
func appendhandle(b []byte, req *request) []byte {
	return appendresp(b, "200 OK", "", res)
}

// appendresp will append a valid http response to the provide bytes.
// The status param should be the code plus text such as "200 OK".
// The head parameter should be a series of lines ending with "\r\n" or empty.
func appendresp(b []byte, status, head, body string) []byte {
	b = append(b, "HTTP/1.1"...)
	b = append(b, ' ')
	b = append(b, status...)
	b = append(b, '\r', '\n')
	b = append(b, "Server: yaev\r\n"...)
	b = append(b, "Date: "...)
	b = time.Now().AppendFormat(b, "Mon, 02 Jan 2006 15:04:05 GMT")
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, "Content-Length: "...)
		b = strconv.AppendInt(b, int64(len(body)), 10)
		b = append(b, '\r', '\n')
	}
	b = append(b, head...)
	b = append(b, '\r', '\n')
	if len(body) > 0 {
		b = append(b, body...)
	}
	return b
}

// parsereq is a very simple http request parser. This operation
// waits for the entire payload to be buffered before returning a
// valid request.
func parsereq(data []byte, req *request) (leftover []byte, err error) {
	sdata := string(data)
	var i, s int
	var top string
	var clen int
	var q = -1
	// method, path, proto line
	for ; i < len(sdata); i++ {
		if sdata[i] == ' ' {
			req.method = sdata[s:i]
			for i, s = i+1, i+1; i < len(sdata); i++ {
				if sdata[i] == '?' && q == -1 {
					q = i - s
				} else if sdata[i] == ' ' {
					if q != -1 {
						req.path = sdata[s:q]
						req.query = req.path[q+1 : i]
					} else {
						req.path = sdata[s:i]
					}
					for i, s = i+1, i+1; i < len(sdata); i++ {
						if sdata[i] == '\n' && sdata[i-1] == '\r' {
							req.proto = sdata[s:i]
							i, s = i+1, i+1
							break
						}
					}
					break
				}
			}
			break
		}
	}
	if req.proto == "" {
		return data, fmt.Errorf("malformed request")
	}
	top = sdata[:s]
	for ; i < len(sdata); i++ {
		if i > 1 && sdata[i] == '\n' && sdata[i-1] == '\r' {
			line := sdata[s : i-1]
			s = i + 1
			if line == "" {
				req.head = sdata[len(top)+2 : i+1]
				i++
				if clen > 0 {
					if len(sdata[i:]) < clen {
						break
					}
					req.body = sdata[i : i+clen]
					i += clen
				}
				return data[i:], nil
			}
			if strings.HasPrefix(line, "Content-Length:") {
				n, err := strconv.ParseInt(strings.TrimSpace(line[len("Content-Length:"):]), 10, 64)
				if err == nil {
					clen = int(n)
				}
			}
		}
	}
	// not enough data
	return data, nil
}
