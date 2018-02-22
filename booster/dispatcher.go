package booster

import (
	"context"
	"errors"
	"log"
	"net"
	"time"

	"github.com/danielmorandini/booster-network/socks5"
	"golang.org/x/net/proxy"
)

type LoadBalancer interface {
	GetNodeBalanced(exep ...string) (Node, error)
	AddTunnel(node Node, target net.Addr)
	RemoveTunnel(node Node, target net.Addr) error
}

// FallbackDialer combines DialContext and Dial methods.
type FallbackDialer interface {
	Dialer
	proxy.Dialer
}

type Dispatcher struct {
	*log.Logger
	LoadBalancer

	Fallback FallbackDialer
}

// NewBalancedDialer returns a Dispatcher instance.
func NewDialer(balancer LoadBalancer) *Dispatcher {
	d := new(Dispatcher)
	d.LoadBalancer = balancer

	d.Fallback = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	return d
}

func (d *Dispatcher) nodeFinderFunc() func() (Node, error) {
	var ids []string

	return func() (Node, error) {
		if len(ids) > 0 {
			d.Printf("dialer: dialed with nodes: %+v", ids)
		}

		n, err := d.GetNodeBalanced(ids...)
		if err == nil {
			ids = append(ids, n.ID())
		}
		return n, err
	}
}

func (d *Dispatcher) dialerForNode(node Node) (socks5.Dialer, error) {
	if node.IsLocal() {
		d.Printf("dialer: using local gateway")
		return d.Fallback, nil
	}

	d.Printf("dialer: using SOCKS5 gateway @ %v", node.ProxyAddr().String())
	return newSocks5Dialer(d.Fallback, node.ProxyAddr().Network(), node.ProxyAddr().String())
}

// DialContext uses the underlying load balancer to retrieve a possibile socks5 proxy
// address to chain the connection to. If none available, dials the connection using
// the default net.Dialer.
func (d *Dispatcher) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	nff := d.nodeFinderFunc()
	node, err := nff()
	if err != nil {
		return nil, errors.New("dialer: " + err.Error())
	}

	// trace number of iterations
	i := 0
	for {
		i++
		d.Printf("dialer: iteration (%v): to %v", i, addr)

		target, err := net.ResolveTCPAddr(network, addr)
		if err != nil {
			return nil, errors.New("dialer: unable to create addr: " + err.Error())
		}

		// first get a dialer
		dialer, err := d.dialerForNode(node)
		if err != nil {
			return nil, errors.New("dialer: " + err.Error())
		}

		// try to get a connection
		d.AddTunnel(node, target)
		conn, cerr := dialer.DialContext(ctx, network, addr)
		if cerr == nil {
			return conn, cerr
		}

		// remove the tunnel the we've created.
		if err = d.RemoveTunnel(node, target); err != nil {
			return nil, errors.New("dialer: " + err.Error())
		}

		// simply return if it was a context error
		if cerr == ctx.Err() {
			return nil, err
		}

		// in case of a connection error, try with another node if possible.
		// otherwise, return the last connection error that we got back.
		node, err = nff()
		if err != nil {
			if cerr == nil {
				cerr = err
			}
			return nil, cerr
		}
	}
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
