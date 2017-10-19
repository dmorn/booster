// Package proxy provides a transparent http forward proxy implementation
package proxy

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

// Proxy is a Forward Proxy.
type Proxy struct {
	transport *http.Transport

	port int
}

// NewProxy creates a new Proxy with custom http.Transport. `port` is the port that
// the proxy will be listening on.
func NewProxy(port int) *Proxy {
	t := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
		MaxIdleConns:        100,
		IdleConnTimeout:     5 * time.Second,
		DisableKeepAlives:   false,
	}

	return &Proxy{
		transport: t,
		port:      port,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CONNECT is used when the client wants to setup a TLS.
	// not supported yet
	if r.Method == http.MethodConnect {
		http.Error(w, http.ErrNotSupported.ErrorString, http.StatusInternalServerError)
		return
	}

	c := http.Client{
		Transport: p.transport,
	}

	// Check `send` in net/http/client.go
	r.RequestURI = ""

	resp, err := c.Do(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)
}

// ListenAndServe listens on port 3128 by default.
func (p *Proxy) ListenAndServe() error {
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(p.port),
		Handler: p,
	}

	return srv.ListenAndServe()
}
