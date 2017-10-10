package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
)

func newDirector(r *http.Request) func(*http.Request) {
	return func(req *http.Request) {
		req.URL.Host = r.Host

		reqLog, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			log.Printf("Got error %s\n %+v\n", err.Error(), req)
		}

		log.Println(string(reqLog))
	}
}

type Proxy struct {
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proxy := &httputil.ReverseProxy{
		Transport: &http.Transport{},
		Director:  newDirector(r),
	}
	proxy.ServeHTTP(w, r)
}
