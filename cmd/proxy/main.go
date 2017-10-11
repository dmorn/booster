package main

import (
	"flag"
	"log"
	"strconv"

	"github.com/danielmorandini/booster/proxy"
)

var (
	port = flag.Int("port", 3128, "Proxy listening port")
	crt  = flag.String("cert", "server.crt", ".crt file path")
	key  = flag.String("key", "server.key", ".key file path")
)

func main() {
	flag.Parse()

	p, err := proxy.NewProxy(strconv.Itoa(*port), *crt, *key)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Proxy listening on port :%s", strconv.Itoa(*port))
	log.Fatal(p.ListenAndServe())
}
