package node

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
	"golang.org/x/net/proxy"
)

// LoadBalancer is a wrapper around the GetNodeBalanced and CloseNode functions.
type LoadBalancer interface {
	// GetNodeBalanced should returns a node id, using internally a
	// balancing algorithm.
	GetNodeBalanced(exp ...string) (*Node, error)
}

// Proxy is a SOCK5 server.
type Proxy struct {
	*socks5.Socks5
}

// NewProxy returns a new proxy instance.
func NewProxy(dialer socks5.Dialer, log *log.Logger, ps PubSub) *Proxy {
	p := new(Proxy)
	p.Socks5 = socks5.NewSOCKS5(dialer, log, ps)

	return p
}

// NewProxyBalancer returns a new proxy instance. balancer is passed as
// a paramenter to the dialer that the proxy will use.
// balancer will be used by the proxy dialer to fetch the
// proxy addresses that can be chained to this proxy.
func NewProxyBalancer(balancer LoadBalancer, ps PubSub) *Proxy {
	d := NewDialer(balancer)
	log := log.New(os.Stdout, "PROXY   ", log.LstdFlags)
	p := NewProxy(d, log, ps)
	d.Logger = log

	return p
}

type socks5Dialer struct {
	dialer proxy.Dialer
}

func newSocks5Dialer(forward proxy.Dialer, network, addr string) (*socks5Dialer, error) {
	sd := new(socks5Dialer)
	dialer, err := proxy.SOCKS5(network, addr, nil, forward)
	if err != nil {
		return nil, err
	}

	sd.dialer = dialer

	return sd, nil
}

func (d *socks5Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	errc := make(chan error)
	connc := make(chan net.Conn)

	go func() {
		conn, err := d.dialer.Dial(network, address)
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

// Dialer implements the DialContext method.
type Dialer struct {
	*log.Logger
	LoadBalancer

	Fallback FallbackDialer
}

// FallbackDialer combines DialContext and Dial methods.
type FallbackDialer interface {
	socks5.Dialer
	proxy.Dialer
}

// NewDialer returns a Dialer instance.
func NewDialer(balancer LoadBalancer) *Dialer {
	d := new(Dialer)
	d.LoadBalancer = balancer

	d.Fallback = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	return d
}

func (d *Dialer) nodeFinderFunc() func() (*Node, error) {
	var ids []string

	return func() (*Node, error) {
		n, err := d.GetNodeBalanced(ids...)
		if err != nil {
			ids = append(ids, n.ID())
		}
		return n, err
	}
}

func (d *Dialer) dialerForNode(node *Node) (socks5.Dialer, error) {
	if node.isLocal {
		d.Printf("dialer: using local gateway")
		return d.Fallback, nil
	}

	d.Printf("dialer: using SOCKS5 gateway @ %v", node.PAddr.String())
	return newSocks5Dialer(d.Fallback, node.PAddr.Network(), node.PAddr.String())
}


// DialContext uses the underlying load balancer to retrieve a possibile socks5 proxy
// address to chain the connection to. If none available, dials the connection using
// the default net.Dialer.
func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	nff := d.nodeFinderFunc()
	node, err := nff()
	if err != nil {
		return nil, errors.New("dialer: " + err.Error())
	}

	for {
		// first get a dialer
		dialer, err := d.dialerForNode(node)
		if err != nil {
			return nil, errors.New("dialer: " + err.Error())
		}

		// try to get a connection
		conn, cerr := dialer.DialContext(ctx, network, addr)
		if cerr == nil {
			return conn, cerr
		}

		// simply return if it was a context error
		if cerr == ctx.Err() {
			return nil, err
		}

		// in case of a connection error, try with another node if possible.
		// otherwise, return the last connection error that we got back.
		node, err = nff()
		if err != nil {
			return nil, cerr
		}
	}
}
