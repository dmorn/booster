package main

import (
	"flag"
	"log"

	"github.com/danielmorandini/booster/proxy"
)

var (
	port = flag.Int("port", 3128, "Proxy listening port")
)

func main() {
	flag.Parse()

	p := proxy.NewProxy(*port)
	log.Printf("Proxy listening on port :%v", *port)
	log.Fatal(p.ListenAndServe())
}
