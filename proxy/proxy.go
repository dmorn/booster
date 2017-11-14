package proxy

import (
	"context"
	"net"

	"github.com/danielmorandini/booster/socks5"
	"golang.org/x/net/proxy"
)

var p1 string = "booster-pi1:1080"

type Proxy struct {
	*socks5.Socks5
}

func NewProxyServer(port int) *Proxy {
	p := new(Proxy)
	p.Socks5 = new(socks5.Socks5)
	p.Dialer = new(dialer)

	return p
}

type dialer struct {
}

func (d *dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	socksDialer, err := proxy.SOCKS5("tcp", p1, nil, new(net.Dialer))
	if err != nil {
		return nil, err
	}

	errc := make(chan error, 1)
	connc := make(chan net.Conn, 1)

	go func() {
		conn, err := socksDialer.Dial(network, addr)
		if err != nil {
			errc <- err
			return
		}

		connc <- conn
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errc:
		return nil, err
	case conn := <-connc:
		return conn, nil
	}
}
