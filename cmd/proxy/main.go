package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster/proxy"
)

var (
	port = flag.Int("port", 1080, "SOCKS listening port")
)

func main() {
	flag.Parse()

	l := log.New(os.Stdout, "BOOSTER ", log.LstdFlags)
	s := &proxy.Server{Port: *port, Log: l}

	log.Printf("Proxy Server listening on port :%v", *port)
	log.Fatal(s.ListenAndServe())
}
