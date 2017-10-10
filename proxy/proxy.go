package proxy

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// Proxy is actually an HTTP (not HTTPS atm) forward proxy.
type Proxy struct {
	transport *http.Transport

	// server config
	port string

	// TLS .crt and .key file paths
	crt string
	key string
}

// NewProxy creates a new Proxy with custom http.Transport. `port` is the port that
// the proxy will be listening on. `crt` and `key` are used to configure TLS.
func NewProxy(port, crt, key string) (*Proxy, error) {
	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &Proxy{
		transport: t,
		port:      port,
		crt:       crt,
		key:       key,
	}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxy := &httputil.ReverseProxy{
		Transport: p.transport,
		Director:  p.newDirector(r),
	}
	proxy.ServeHTTP(w, r)
}

// ListenAndServeTLS is supposed to be used for HTTPS forward proxy.
// does not work yet.
func (p *Proxy) ListenAndServeTLS() error {
	srv := &http.Server{
		Addr:    ":3124",
		Handler: p,
	}

	return srv.ListenAndServeTLS(p.crt, p.key)
}

// ListenAndServe accepts HTTP connections.
func (p *Proxy) ListenAndServe() error {
	srv := &http.Server{
		Addr:    ":" + p.port,
		Handler: p,
	}

	return srv.ListenAndServe()
}

func (p *Proxy) newDirector(r *http.Request) func(*http.Request) {
	return func(req *http.Request) {
		req.URL.Host = r.Host
		req.URL.Scheme = "http"

		reqLog, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			log.Printf("Got error %s\n %+v\n", err.Error(), req)
		}

		log.Println(string(reqLog))
	}
}
