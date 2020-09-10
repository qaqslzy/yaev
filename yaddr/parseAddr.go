package yaddr

import "strings"

/**
*
* @author Liu Weiyi
* @date 2020/8/4 4:06 下午
 */

type AddrOpts struct {
	ReusePort bool
}

func ParseAddr(addr string) (network, address string, opts AddrOpts, stdlib bool) {
	network = "tcp"
	address = addr
	opts.ReusePort = false
	if strings.Contains(address, "://") {
		network = strings.Split(address, "://")[0]
		address = strings.Split(address, "://")[1]
	}
	if strings.HasSuffix(network, "-net") {
		stdlib = true
		network = network[:len(network)-4]
	}
	q := strings.Index(address, "?")
	if q != -1 {
		for _, part := range strings.Split(address[q+1:], "&") {
			kv := strings.Split(part, "=")
			if len(kv) == 2 {
				switch kv[0] {
				case "reuseport":
					if len(kv[1]) != 0 {
						switch kv[1][0] {
						default:
							opts.ReusePort = kv[1][0] >= '1' && kv[1][0] <= '9'
						case 'T', 't', 'Y', 'y':
							opts.ReusePort = true

						}
					}
				}
			}
		}
		address = address[:q]
	}
	return
}