package proxy

import (
	"container/ring"
	"context"
	"fmt"
	"net"

	"github.com/danielmorandini/booster/socks5"
	"golang.org/x/net/proxy"
)

var connectedProxies = []string{"booster-pi1:1080", "booster-pi1:1081"}

type Proxy struct {
	*socks5.Socks5
}

func NewProxyServer(port int) *Proxy {
	p := new(Proxy)
	p.Socks5 = new(socks5.Socks5)

	d := new(dialer)
	d.ring = ring.New(len(connectedProxies))
	for _, v := range connectedProxies {
		d.ring.Value = v
		d.ring = d.ring.Next()
	}

	p.Dialer = d

	c := make(chan int)
	if err := p.RegisterWorkloadListener("fooid", c); err != nil {
		panic("unable to register status listener")
	}

	go func() {
		for workload := range c {
			p.Printf("[PROXY workload]: %v\n", workload)
		}
	}()

	return p
}

type dialer struct {
	ring *ring.Ring
}

func (d *dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	paddr := d.ring.Value.(string)
	d.ring = d.ring.Next()

	fmt.Printf("Proxy Address: %v\n", paddr)
	socksDialer, err := proxy.SOCKS5("tcp", paddr, nil, new(net.Dialer))
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
