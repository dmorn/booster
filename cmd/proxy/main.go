package main

import (
	"log"
	"net/http"

	"github.com/danielmorandini/booster/proxy"
)

func main() {
	p := &proxy.Proxy{}
	http.Handle("/", p)
	if err := http.ListenAndServe(":3128", nil); err != nil {
		log.Fatal(err)
	}
}
