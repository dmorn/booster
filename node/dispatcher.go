package node

import (
	"context"
	"errors"
	"log"
	"os"
	"net"
	"time"

	"github.com/danielmorandini/booster/socks5"
	"golang.org/x/net/proxy"
)

type GetNodesFunc func() (*Node, []*Node)

// FallbackDialer combines DialContext and Dial methods.
type FallbackDialer interface {
	socks5.Dialer
	proxy.Dialer
}

// Dispatcher implements the DialContext method.
type Dispatcher struct {
	*log.Logger

	Nodes GetNodesFunc

	Fallback FallbackDialer
}

// NewDialer returns a Dialer instance.
func NewDispatcher(f GetNodesFunc) *Dispatcher {
	d := new(Dispatcher)
	d.Logger = log.New(os.Stdout, "DISPTCR  ", log.LstdFlags)
	d.Nodes = f

	d.Fallback = &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	return d
}

func (d *Dispatcher) nodeFinderFunc() func() (*Node, error) {
	var ids []string

	return func() (*Node, error) {
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

// GetNodeBalanced collects the workload of its registered nodes,
// and compares them to the workload of the root node.
//
// Returns an error if no candidate is found, either because
// none was provided or because no entry's workload was under
// the treshold.
//
// exp is a list of ids, which are considered as nodes that should
// not be taken into consideration.
func (d *Dispatcher) GetNodeBalanced(exp ...string) (*Node, error) {
	root, nodes := d.Nodes()
	tr := root.Workload()

	var c *Node // candidate entry
	var twl int // total workload

	for _, e := range nodes {
		// do not condider non active nodes
		if !e.IsActive() {
			continue
		}

		ewl := e.Workload()
		twl += ewl

		// check if node is in the exceptions
		if isIn(e.ID(), exp...) {
			continue
		}

		if c == nil {
			c = e
		}

		cwl := c.Workload() // candidate workload

		if ewl < cwl {
			c = e
		}
	}

	// we did not find any suitable node
	if c == nil {
		if isIn(root.ID(), exp...) {
			return root, errors.New("balancer: no suitable node found")
		}

		return root, nil
	}

	// tr is the sum of the local workload and the remote node's workload.
	// this is why we have to subtract the total remote workload to understand
	// how the load on this node is.
	if c.Workload() > (tr - twl) {
		// return the candidate even if the local node is the most suitable one
		if isIn(root.ID(), exp...) {
			return c, nil
		}

		return root, nil
	}

	return c, nil
}

func isIn(id string, ids ...string) bool {
	for _, v := range ids {
		if id == v {
			return true
		}
	}
	return false
}

func (d *Dispatcher) dialerForNode(node *Node) (socks5.Dialer, error) {
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
func (d *Dispatcher) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	nff := d.nodeFinderFunc()
	node, err := nff()
	if err != nil {
		return nil, errors.New("dialer: " + err.Error())
	}

	// trace number of iterations
	i := 0
	for {
		i += 1
		d.Printf("dialer: iteration (%v): to %v", i, addr)

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
