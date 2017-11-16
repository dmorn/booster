package booster

import (
	"context"
	"net"

	"github.com/danielmorandini/booster/socks5"
	"golang.org/x/net/proxy"
)

type Proxy struct {
	*socks5.Socks5
}

func NewProxy(balancer *Balancer) *Proxy {
	p := new(Proxy)
	p.Socks5 = new(socks5.Socks5)

	d := new(dialer)
	d.balancer = balancer
	p.Dialer = d

	return p
}

type dialer struct {
	balancer *Balancer
}

func (d *dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var socksDialer proxy.Dialer

	if paddr, err := d.balancer.GetProxy(); err != nil {
		socksDialer = new(net.Dialer) // just a normal dialer
	} else {
		socksDialer, err = proxy.SOCKS5("tcp", paddr, nil, new(net.Dialer))
		if err != nil {
			return nil, err
		}
	}

	ec := make(chan error, 1)
	cc := make(chan net.Conn, 1)

	go func() {
		conn, err := d.dialFallback(ctx, socksDialer, network, addr)
		if err != nil {
			ec <- err
			return
		}

		cc <- conn
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-ec:
		return nil, err
	case conn := <-cc:
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
