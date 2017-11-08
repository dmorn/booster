package main

import (
	"flag"
	"log"
	"os"

	"github.com/danielmorandini/booster/socks5"
)

var (
	port = flag.Int("port", 1080, "SOCKS listening port")
)

func main() {
	flag.Parse()

	proxy := new(socks5.Socks5)
	proxy.Port = *port
	proxy.Log = log.New(os.Stdout, "BOOSTER ", log.LstdFlags)

	log.Printf("Proxy Server listening on port :%v", *port)
	log.Fatal(proxy.ListenAndServe())
}
