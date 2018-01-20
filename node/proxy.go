package node

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
	"golang.org/x/net/proxy"
)

// LoadBalancer is a wrapper around the GetNodeBalanced function.
type LoadBalancer interface {
	// GetNodeBalanced should returns a node id, using internally a
	// balancing algorithm.
	// tr should be used to set a minimum treshold requirement.
	GetNodeBalanced(tr int) (*RemoteNode, error)
	CloseNode(id string) (*RemoteNode, error)
}

// Proxy is a SOCK5 server.
type Proxy struct {
	*socks5.Socks5
}

// NewProxy returns a new proxy instance.
func NewProxy(dialer socks5.Dialer, log *log.Logger) *Proxy {
	p := new(Proxy)
	p.Socks5 = socks5.NewSOCKS5(dialer, log)

	return p
}

// NewProxyBalancer returns a new proxy instance. balancer is passed as
// a paramenter to the dialer that the proxy will use.
// balancer will be used by the proxy dialer to fetch the
// proxy addresses that can be chained to this proxy.
func NewProxyBalancer(balancer LoadBalancer, tracer Tracer) *Proxy {
	d := NewDialer(balancer, tracer)
	log := log.New(os.Stdout, "PROXY   ", log.LstdFlags)
	p := NewProxy(d, log)
	d.Logger = log

	// keep track of local proxy usage
	c := p.Sub(socks5.TopicWorkload)
	go func() {
		defer func() {
			p.Unsub(c, socks5.TopicWorkload)
		}()

		for i := range c {
			d.Lock()
			wm, ok := i.(socks5.WorkloadMessage)
			if !ok {
				p.Printf("proxy: unable to recognise workload message: %v", wm)
				return
			}

			d.workload = wm.Load
			d.Unlock()
		}
	}()

	return p
}

// Dialer implements the DialContext method.
type Dialer struct {
	*log.Logger
	Tracer
	LoadBalancer
	Fallback FallbackDialer

	sync.Mutex
	// local proxy workload.
	// Be careful that this value will be updated each time that the underlying
	// socks5 proxy is tunneling some data, so it is updated either when
	// we directly dial with the remote host AND when we chain with other proxies.
	workload int
}

// FallbackDialer combines DialContext and Dial methods.
type FallbackDialer interface {
	socks5.Dialer
	proxy.Dialer
}

// NewDialer returns a Dialer instance.
func NewDialer(balancer LoadBalancer, tracer Tracer) *Dialer {
	d := new(Dialer)
	d.LoadBalancer = balancer
	d.Tracer = tracer
	d.Fallback = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	return d
}

// DialContext uses the underlying load balancer to retrieve a possibile socks5 proxy
// address to chain the connection to. If none available, dials the connection using
// the default net.Dialer.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	d.Lock()
	lwl := d.workload // local workload
	d.Unlock()

	node, err := d.GetNodeBalanced(lwl)
	if err != nil {
		d.Printf("dialer: dialing directly: %v", err)
		return d.Fallback.DialContext(ctx, network, addr)
	}

	paddr := net.JoinHostPort(node.Host, node.Pport)
	ec := make(chan error, 1)
	cc := make(chan net.Conn, 1)

	go func() {
		d.Printf("dialer: using SOCKS5 gateway @ %v", paddr)

		socksDialer, err := proxy.SOCKS5(network, paddr, nil, d.Fallback)
		if err != nil {
			ec <- err
			return
		}

		conn, err := socksDialer.Dial(network, addr)
		if err != nil {
			// the node that we tried to chain to is down or unusable.
			// fallback to a normal dialer and close this node.
			d.Printf("dialer: unable to Dial using gateway @ %v. Fallback", node.ID())
			if _, err := d.CloseNode(node.ID()); err != nil {
				d.Printf("dialer: unable to close node (%v)", node.ID())
			}
			if d.Tracer != nil {
				d.Trace(node)
			}

			conn, err = d.Fallback.Dial(network, addr)
			if err != nil {
				ec <- err
				return
			}
		}

		cc <- conn
	}()

	select {
	case <-ctx.Done():
		return nil, errors.New("dialer: dial context: " + ctx.Err().Error())
	case err := <-ec:
		return nil, err
	case conn := <-cc:
		return conn, nil
	}
}
