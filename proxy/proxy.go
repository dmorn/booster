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

	c := make(chan uint8)
	if err := p.RegisterStatusListener(p1, c); err != nil {
		panic("unable to register status listener")
	}

	go func() {
		for status := range c {
			switch status {
			case 0:
				p.Printf("[PROXY status]: IDLE")
			case 1:
				p.Printf("[PROXY status]: proxying")
			}
		}
	}()

	return p
}

type dialer struct {
}

func (d *dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// p1 should be received from booster
	socksDialer, err := proxy.SOCKS5("tcp", p1, nil, new(net.Dialer))
	if err != nil {
		return nil, err
	}

	errc := make(chan error, 1)
	connc := make(chan net.Conn, 1)

	go func() {
		conn, err := d.dialFallback(ctx, socksDialer, network, addr)
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

func (d *dialer) dialFallback(ctx context.Context, socksDialer proxy.Dialer, network, addr string) (net.Conn, error) {
	conn, err := socksDialer.Dial(network, addr)
	if err == nil {
		return conn, err
	}

	// try without proxy
	fallback := new(net.Dialer)
	return fallback.DialContext(ctx, network, addr)
}
