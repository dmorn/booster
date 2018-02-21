package proxy

// Dialer is the interface that wraps the DialContext function.
type Dialer interface {
	// DialContext opens a connection to addr, which should
	// be a canonical address with host and port.
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

type Proxy interface {
	SetDialer(dialer Dialer)
	NotifyTunnel() (<-chan interface{}, error)
	ListenAndServe(ctx contenxt.Context, port int)
}
