package node

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/danielmorandini/booster-network/socks5"
	"golang.org/x/net/proxy"
)

type Proxy struct {
	*socks5.Socks5
}

func NewProxy(dialer socks5.Dialer, log *log.Logger) *Proxy {
	p := new(Proxy)
	p.Socks5 = socks5.NewSOCKS5(dialer, log)

	return p
}

func Proxy(balancer LoadBalancer) *Proxy {
	d := NewDialer(balancer)
	log := log.New(os.Stdout, "PROXY   ", log.LstdFlags)
	return NewProxy(d, log)
}

type Dialer struct {
	balancer LoadBalancer
}

func NewDialer(balancer LoadBalancer) *Dialer {
	d := new(dialer)
	d.balancer = balancer

	return d
}

func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var socksDialer proxy.Dialer

	if paddr, err := d.balancer.GetProxy(); err != nil {
		socksDialer = new(net.Dialer) // just a normal Dialer
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

func (d *Dialer) dialFallback(ctx context.Context, socksDialer proxy.Dialer, network, addr string) (net.Conn, error) {
	conn, err := socksDialer.Dial(network, addr)
	if err == nil {
		return conn, err
	}

	// try without proxy
	fallback := new(net.Dialer)
	return fallback.DialContext(ctx, network, addr)
}
